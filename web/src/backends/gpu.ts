// WebGL2 backend. This is the *accelerator* seam, not a full GPU port: it
// runs the effects that map cleanly to a fragment shader on the GPU and
// hands everything else to the Go/WASM core on this same thread. Right now
// only "pixelate" has a shader (proof of the seam); `accelerates()` is the
// honest list, and backend.ts / main.ts route non-accelerated effects to a
// CPU backend.
//
// Effects that will never live here: Floyd–Steinberg dithering (serial
// error diffusion) and ASCII (glyph rasterisation). Those stay CPU by
// design — see CLAUDE.md.

import {
  initWasm,
  applyFilter as wasmApplyFilter,
  asciiText as wasmAsciiText,
  renderGIF as wasmRenderGIF,
  renderGradient as wasmRenderGradient,
  gradientCSS as wasmGradientCSS,
  renderNoiseFieldGIF as wasmRenderNoiseFieldGIF,
  renderGenerator as wasmRenderGenerator,
  extractPalette as wasmExtractPalette,
  genPalette as wasmGenPalette,
} from "../wasm";
import { registerBackend, type FilterBackend } from "../backend";
import { blitCanvas } from "../canvas";
import type {
  FilterParams,
  GifKeyframe,
  GifOptions,
  GradientParams,
  GradientCSSOptions,
  NoiseFieldParams,
  GeneratorParams,
  PaletteExtractOptions,
  PaletteHarmonyOptions,
} from "../wasm";

const ACCELERATED = new Set(["pixelate"]);

const VERT = `#version 300 es
precision highp float;
const vec2 verts[3] = vec2[3](vec2(-1.0, -1.0), vec2(3.0, -1.0), vec2(-1.0, 3.0));
out vec2 v_uv;
void main() {
  vec2 p = verts[gl_VertexID];
  v_uv = (p + 1.0) * 0.5;
  gl_Position = vec4(p, 0.0, 1.0);
}`;

// Snap the sample point to the centre of its block and read a mip level
// roughly the block's size, so one tap approximates the CPU block average.
// Clip-space y=-1 (framebuffer bottom) maps to texture t=0 (first ImageData
// row), so readPixels — which returns bottom row first — comes back in
// ImageData order with no manual flip.
const FRAG_PIXELATE = `#version 300 es
precision highp float;
uniform sampler2D u_tex;
uniform vec2 u_resolution;
uniform float u_block;
in vec2 v_uv;
out vec4 o_color;
void main() {
  vec2 texel = v_uv * u_resolution;
  vec2 centre = (floor(texel / u_block) + 0.5) * u_block;
  float lod = max(0.0, log2(u_block));
  o_color = textureLod(u_tex, centre / u_resolution, lod);
}`;

// GLSL ES 3.00 port of Gradient Studio's fragment shader — the GPU twin of
// internal/noisefield. Kept near line-for-line with the Go core so the two
// engines agree: `palette()` interpolates in linear light (like the core,
// not the raw-sRGB original) and texture passes use `fc` (top-left origin)
// so grain/scanline phase survives readback. Organic + graded fields still
// differ slightly — highp float here vs float64 in Go makes the fbm-driven
// detail drift — so the CPU port stays authoritative for PNG/GIF export.
const FRAG_NOISE = `#version 300 es
precision highp float;
uniform vec2 u_res;
uniform float u_time;
uniform int u_type;
uniform int u_genre;
uniform int u_texture;
uniform float u_angle;
uniform vec3 u_stops[16];
uniform int u_nstops;
uniform vec3 u_spotCol[8];
uniform vec2 u_spotPos[8];
uniform int u_nspots;
uniform float u_freq;
uniform float u_warp;
uniform float u_seed;
out vec4 o_color;
#define PI 3.14159265
vec3 mod289(vec3 x){return x-floor(x*(1.0/289.0))*289.0;}
vec2 mod289(vec2 x){return x-floor(x*(1.0/289.0))*289.0;}
vec3 permute(vec3 x){return mod289(((x*34.0)+1.0)*x);}
float snoise(vec2 v){
  const vec4 C=vec4(0.211324865405187,0.366025403784439,-0.577350269189626,0.024390243902439);
  vec2 i=floor(v+dot(v,C.yy));vec2 x0=v-i+dot(i,C.xx);
  vec2 i1=(x0.x>x0.y)?vec2(1.0,0.0):vec2(0.0,1.0);
  vec4 x12=x0.xyxy+C.xxzz;x12.xy-=i1;i=mod289(i);
  vec3 p=permute(permute(i.y+vec3(0.0,i1.y,1.0))+i.x+vec3(0.0,i1.x,1.0));
  vec3 m=max(0.5-vec3(dot(x0,x0),dot(x12.xy,x12.xy),dot(x12.zw,x12.zw)),0.0);m=m*m;m=m*m;
  vec3 x=2.0*fract(p*C.www)-1.0;vec3 h=abs(x)-0.5;vec3 ox=floor(x+0.5);vec3 a0=x-ox;
  m*=1.79284291400159-0.85373472095314*(a0*a0+h*h);
  vec3 g;g.x=a0.x*x0.x+h.x*x0.y;g.yz=a0.yz*x12.xz+h.yz*x12.yw;
  return 130.0*dot(m,g);
}
float fbm(vec2 p){float v=0.0,a=0.5;for(int i=0;i<3;i++){v+=a*snoise(p);p=p*2.03+vec2(1.7,9.2);a*=0.5;}return v*0.5+0.5;}
float hash(vec2 p){return fract(sin(dot(p,vec2(12.9898,78.233)))*43758.5453);}
vec3 hsv2rgb(vec3 c){vec4 K=vec4(1.0,2.0/3.0,1.0/3.0,3.0);vec3 p=abs(fract(c.xxx+K.xyz)*6.0-K.www);return c.z*mix(K.xxx,clamp(p-K.xxx,0.0,1.0),c.y);}
vec3 hueShift(vec3 c,float h){const vec3 k=vec3(0.57735);float cs=cos(h),sn=sin(h);return c*cs+cross(k,c)*sn+k*dot(k,c)*(1.0-cs);}
float lum(vec3 c){return dot(c,vec3(0.299,0.587,0.114));}
vec3 srgbToLinear(vec3 c){return mix(c/12.92,pow((c+0.055)/1.055,vec3(2.4)),step(0.04045,c));}
vec3 linearToSrgb(vec3 c){return mix(c*12.92,1.055*pow(max(c,0.0),vec3(1.0/2.4))-0.055,step(0.0031308,c));}
vec3 palette(float t){
  t=clamp(t,0.0,1.0);float f=t*float(u_nstops-1);vec3 c=srgbToLinear(u_stops[0]);
  for(int i=0;i<15;i++){if(i<u_nstops-1){float lo=float(i);if(f>=lo&&f<=lo+1.0)c=mix(srgbToLinear(u_stops[i]),srgbToLinear(u_stops[i+1]),f-lo);}}
  return linearToSrgb(c);
}
void main(){
  vec2 fc=vec2(gl_FragCoord.x,u_res.y-gl_FragCoord.y); // top-left origin, matches the Go port
  vec2 uv=fc/u_res;vec2 p=vec2(uv.x,1.0-uv.y);
  float aspect=u_res.x/u_res.y;float minDim=min(u_res.x,u_res.y);
  vec2 dir=vec2(sin(u_angle),-cos(u_angle));float t=u_time;
  if(u_texture==3){float amp=max(6.0,minDim*0.016)/u_res.x;float f=2.0*PI/max(180.0/u_res.y,0.34);p.x+=sin(p.y*f)*amp+sin(p.y*f*0.46)*amp*0.45;}
  else if(u_texture==4){float amp=max(4.0,minDim*0.011)/u_res.x;float f=2.0*PI/max(140.0/u_res.y,0.28);p.x+=sin(p.y*f)*amp+sin(p.y*f*1.9)*amp*0.35;}
  vec2 px=(p-0.5)*u_res;vec3 col;
  if(u_type<=4){
    float tt=0.0;
    if(u_type==0||u_type==3){float len=abs(dir.x)*u_res.x*0.5+abs(dir.y)*u_res.y*0.5;tt=dot(px,dir)/len*0.5+0.5;}
    else if(u_type==1){tt=length(px)/(length(u_res)*0.5);}
    else if(u_type==2){float a=atan(px.x,-px.y);tt=fract((a-u_angle)/(2.0*PI));}
    else{tt=(abs(px.x)/(u_res.x*0.5)+abs(px.y)/(u_res.y*0.5))*0.5;}
    col=palette(tt);
  }else{
    vec2 pa=vec2(p.x*aspect,p.y);vec2 q=pa;
    vec2 np=(q+dir*t*0.03)*u_freq*0.75+u_seed;
    vec2 w1=vec2(fbm(np+t*0.05),fbm(np+vec2(5.2,1.3)-t*0.04));
    q+=(w1-0.5)*u_warp;
    if(u_type==7){vec2 w2=vec2(fbm(q*u_freq*1.15+3.1+t*0.03),fbm(q*u_freq*1.15+7.7-t*0.02));q+=(w2-0.5)*u_warp*0.55;}
    float pw=u_type==5?2.4:(u_type==6?3.6:2.0);float eps=u_type==7?0.012:(u_type==5?0.006:0.002);
    vec3 acc=vec3(0.0);float ws=0.0;
    for(int i=0;i<8;i++){if(i<u_nspots){vec2 s=vec2(u_spotPos[i].x*aspect,u_spotPos[i].y);float d=distance(q,s);float w=1.0/(pow(d,pw)+eps);acc+=u_spotCol[i]*w;ws+=w;}}
    col=acc/max(ws,1e-6);
    float n=fbm(pa*u_freq*0.8+u_seed*0.37+t*0.02);
    float s=dot(pa-vec2(aspect*0.5,0.5),dir);
    if(u_genre==0){float band=sin((s*1.6+n*0.6)*PI);col*=0.88+0.12*band;col+=pow(max(band,0.0),3.0)*0.12;}
    else if(u_genre==1){col=mix(vec3(lum(col)),col,0.6);float band=sin((s*1.8+n*0.8)*PI);col*=0.76+0.24*band;col+=pow(max(band,0.0),4.0)*0.26;}
    else if(u_genre==2){col=hueShift(col,(n-0.5)*2.2);}
    else if(u_genre==3){vec3 rb=hsv2rgb(vec3(fract(s*1.4+n*0.7+t*0.02),0.55,1.0));col=mix(col,rb,0.4)+0.06;}
    else if(u_genre==4){col=mix(vec3(lum(col)),col,1.4);col=(col-0.5)*1.15+0.5;col+=col*pow(n,3.0)*0.25;}
    else if(u_genre==5){col=mix(col,vec3(1.0),0.42);}
    else if(u_genre==7){vec3 rb=hsv2rgb(vec3(fract(n*1.5+s*0.8),0.75,1.0));col=mix(col,rb,0.65);}
  }
  if(u_texture==1){col+=(hash(fc)-0.5)*0.086;}
  else if(u_texture==2){col=mix(col,vec3(1.0),0.22);col+=(hash(fc)-0.5)*0.039;}
  else if(u_texture==3){col=mix(col,vec3(1.0),0.10);}
  else if(u_texture==4){float st=max(54.0,u_res.x/14.0);float u=fc.x+(u_res.y-fc.y)*0.85+sin(fc.y*0.02)*7.0;float m=mod(u,st);
    float lt=1.0-smoothstep(0.0,1.4,abs(m-0.7));float dk=1.0-smoothstep(0.0,1.2,abs(m-3.5));col+=lt*0.06-dk*0.045;}
  else if(u_texture==5){col+=(hash(fc)-0.5)*0.047;float row=mod(floor(fc.y),4.0);if(row<0.5)col+=0.016;else if(abs(row-2.0)<0.5)col-=0.012;}
  o_color=vec4(clamp(col,0.0,1.0),1.0);
}`;

const NF_FIELD: Record<string, number> = {
  linear: 0, radial: 1, conic: 2, reflected: 3, diamond: 4, mesh: 5, freeform: 6, flow: 7,
};
const NF_STYLE: Record<string, number> = {
  metallic: 0, chrome: 1, iridescent: 2, holographic: 3, neon: 4, pastel: 5, duotone: 6, rainbow: 7,
};
const NF_TEXTURE: Record<string, number> = {
  smooth: 0, grain: 1, frosted: 2, wave: 3, wrinkle: 4, paper: 5,
};
const NF_ORGANIC = new Set(["mesh", "freeform", "flow"]);

function hexToRgb01(hex: string): [number, number, number] {
  const h = hex.replace("#", "");
  const n =
    h.length === 3
      ? parseInt(h[0] + h[0] + h[1] + h[1] + h[2] + h[2], 16)
      : parseInt(h.slice(0, 6), 16);
  return [((n >> 16) & 255) / 255, ((n >> 8) & 255) / 255, (n & 255) / 255];
}

class GpuBackend implements FilterBackend {
  readonly kind = "gpu" as const;

  #gl: WebGL2RenderingContext | null = null;
  #pixelate: WebGLProgram | null = null;
  #noise: WebGLProgram | null = null;
  #vao: WebGLVertexArrayObject | null = null;
  #tex: WebGLTexture | null = null;
  #fbo: WebGLFramebuffer | null = null;
  #fboTex: WebGLTexture | null = null;
  #w = 0;
  #h = 0;

  async init(): Promise<void> {
    const canvas = document.createElement("canvas");
    const gl = canvas.getContext("webgl2", {
      preserveDrawingBuffer: true,
      alpha: false,
      antialias: false,
      powerPreference: "high-performance",
    });
    if (!gl) throw new Error("WebGL2 unavailable");
    this.#gl = gl;
    this.#pixelate = linkProgram(gl, VERT, FRAG_PIXELATE);
    this.#noise = linkProgram(gl, VERT, FRAG_NOISE);
    this.#vao = gl.createVertexArray(); // no attributes; gl_VertexID drives the quad
    this.#tex = gl.createTexture();
    // The CPU core still handles most effects and the text export.
    await initWasm();
  }

  accelerates(name: string): boolean {
    return ACCELERATED.has(name);
  }

  async applyFilter(name: string, img: ImageData, params: FilterParams): Promise<ImageData> {
    if (name === "pixelate") return this.#pixelateGPU(img, params);
    return wasmApplyFilter(name, img, params); // delegated to the CPU core
  }

  async asciiText(img: ImageData, params: FilterParams): Promise<string> {
    return wasmAsciiText(img, params);
  }

  async renderGIF(
    name: string,
    img: ImageData,
    keyframes: GifKeyframe[],
    options: GifOptions,
  ): Promise<Uint8Array> {
    return wasmRenderGIF(name, img, keyframes, options); // multi-frame: CPU core
  }

  // The noise field runs on the GPU here (the point of the accelerator — a
  // smooth animated preview). Gradient sampling and palette work stay on the
  // CPU core; the PNG/GIF exports also use the CPU port for engine-agnostic
  // output.
  async renderGradient(params: GradientParams): Promise<ImageData> {
    return wasmRenderGradient(params);
  }

  async gradientCSS(params: GradientParams, options: GradientCSSOptions): Promise<string> {
    return wasmGradientCSS(params, options);
  }

  async renderNoiseField(params: NoiseFieldParams, w: number, h: number): Promise<ImageData> {
    return this.#noiseGPU(params, w, h);
  }

  renderNoiseFieldToCanvas(
    targetCanvas: HTMLCanvasElement,
    params: NoiseFieldParams,
    w: number,
    h: number,
  ): void {
    const gl = this.#gl!;
    this.#resize(w, h);
    this.#renderNoisePass(params, w, h, null);
    blitCanvas(targetCanvas, gl.canvas as HTMLCanvasElement);
  }

  async renderNoiseFieldGIF(
    start: NoiseFieldParams,
    end: NoiseFieldParams,
    w: number,
    h: number,
    options: GifOptions,
  ): Promise<Uint8Array> {
    return wasmRenderNoiseFieldGIF(start, end, w, h, options); // multi-frame: CPU core
  }

  // Algorithmic generators (attractors, harmonographs, ...) have no shader
  // here; they run on the CPU core like the other non-accelerated work.
  async renderGenerator(
    name: string,
    params: GeneratorParams,
    w: number,
    h: number,
  ): Promise<ImageData> {
    return wasmRenderGenerator(name, params, w, h);
  }

  async extractPalette(img: ImageData, options: PaletteExtractOptions): Promise<string[]> {
    return wasmExtractPalette(img, options);
  }

  async genPalette(options: PaletteHarmonyOptions): Promise<string[]> {
    return wasmGenPalette(options);
  }

  #resize(w: number, h: number): void {
    const gl = this.#gl!;
    if (w === this.#w && h === this.#h && this.#fbo) return;
    this.#w = w;
    this.#h = h;
    gl.canvas.width = w;
    gl.canvas.height = h;

    if (this.#fboTex) gl.deleteTexture(this.#fboTex);
    if (this.#fbo) gl.deleteFramebuffer(this.#fbo);
    this.#fboTex = gl.createTexture();
    gl.bindTexture(gl.TEXTURE_2D, this.#fboTex);
    gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, w, h, 0, gl.RGBA, gl.UNSIGNED_BYTE, null);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST);
    this.#fbo = gl.createFramebuffer();
    gl.bindFramebuffer(gl.FRAMEBUFFER, this.#fbo);
    gl.framebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, this.#fboTex, 0);
  }

  #pixelateGPU(img: ImageData, params: FilterParams): ImageData {
    const gl = this.#gl!;
    const block = Math.max(1, Math.round(Number(params.blockSize ?? 8)));
    this.#resize(img.width, img.height);

    gl.bindTexture(gl.TEXTURE_2D, this.#tex);
    gl.pixelStorei(gl.UNPACK_ALIGNMENT, 1);
    gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, img.width, img.height, 0, gl.RGBA, gl.UNSIGNED_BYTE, img.data);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR);
    gl.generateMipmap(gl.TEXTURE_2D); // mip levels ≈ progressively box-averaged

    gl.bindFramebuffer(gl.FRAMEBUFFER, this.#fbo);
    gl.viewport(0, 0, img.width, img.height);
    gl.useProgram(this.#pixelate);
    gl.bindVertexArray(this.#vao);
    gl.uniform1i(gl.getUniformLocation(this.#pixelate!, "u_tex"), 0);
    gl.uniform2f(gl.getUniformLocation(this.#pixelate!, "u_resolution"), img.width, img.height);
    gl.uniform1f(gl.getUniformLocation(this.#pixelate!, "u_block"), block);
    gl.drawArrays(gl.TRIANGLES, 0, 3);

    const out = new Uint8Array(img.width * img.height * 4);
    gl.readPixels(0, 0, img.width, img.height, gl.RGBA, gl.UNSIGNED_BYTE, out);
    return new ImageData(new Uint8ClampedArray(out.buffer), img.width, img.height);
  }

  #noiseGPU(params: NoiseFieldParams, w: number, h: number): ImageData {
    const gl = this.#gl!;
    this.#resize(w, h);
    this.#renderNoisePass(params, w, h, this.#fbo);
    const out = new Uint8Array(w * h * 4);
    gl.readPixels(0, 0, w, h, gl.RGBA, gl.UNSIGNED_BYTE, out);
    return new ImageData(new Uint8ClampedArray(out.buffer), w, h);
  }

  #renderNoisePass(
    params: NoiseFieldParams,
    w: number,
    h: number,
    targetFbo: WebGLFramebuffer | null,
  ): void {
    const gl = this.#gl!;
    const prog = this.#noise!;

    const organic = NF_ORGANIC.has(params.field);
    const scale = Math.min(100, Math.max(0, params.scale)) / 100;
    const distortion = Math.min(100, Math.max(0, params.distortion)) / 100;
    const warp = distortion * (params.field === "flow" ? 1.1 : params.field === "mesh" ? 0.35 : 0.6);
    const seedInt = Math.trunc(params.seed) || 0;
    const uSeed = (((seedInt % 100000) + 100000) % 100000) + 7.3; // matches noisefield.seedReduce

    const stopsBuf = new Float32Array(48);
    let nStops = 1;
    if (organic) {
      stopsBuf[0] = stopsBuf[1] = stopsBuf[2] = 0.5;
    } else {
      const st = params.stops.length ? params.stops : ["#808080"];
      nStops = Math.min(16, st.length);
      for (let i = 0; i < nStops; i++) {
        const [r, g, b] = hexToRgb01(st[i]);
        stopsBuf[i * 3] = r;
        stopsBuf[i * 3 + 1] = g;
        stopsBuf[i * 3 + 2] = b;
      }
    }

    const colBuf = new Float32Array(24);
    const posBuf = new Float32Array(16);
    let nSpots = 0;
    if (organic) {
      const sp = params.spots.length ? params.spots : [{ color: "#ffffff", x: 0.5, y: 0.5 }];
      nSpots = Math.min(8, sp.length);
      for (let i = 0; i < nSpots; i++) {
        const [r, g, b] = hexToRgb01(sp[i].color);
        colBuf[i * 3] = r;
        colBuf[i * 3 + 1] = g;
        colBuf[i * 3 + 2] = b;
        posBuf[i * 2] = sp[i].x;
        posBuf[i * 2 + 1] = sp[i].y;
      }
    }

    gl.bindFramebuffer(gl.FRAMEBUFFER, targetFbo);
    gl.viewport(0, 0, w, h);
    gl.useProgram(prog);
    gl.bindVertexArray(this.#vao);
    const u = (n: string) => gl.getUniformLocation(prog, n);
    gl.uniform2f(u("u_res"), w, h);
    gl.uniform1f(u("u_time"), params.time || 0);
    gl.uniform1i(u("u_type"), NF_FIELD[params.field] ?? 7);
    gl.uniform1i(u("u_genre"), NF_STYLE[params.style] ?? 6);
    gl.uniform1i(u("u_texture"), NF_TEXTURE[params.texture] ?? 0);
    gl.uniform1f(u("u_angle"), (params.angle * Math.PI) / 180);
    gl.uniform3fv(u("u_stops"), stopsBuf);
    gl.uniform1i(u("u_nstops"), Math.max(1, nStops));
    gl.uniform3fv(u("u_spotCol"), colBuf);
    gl.uniform2fv(u("u_spotPos"), posBuf);
    gl.uniform1i(u("u_nspots"), nSpots);
    gl.uniform1f(u("u_freq"), 3.2 + (0.7 - 3.2) * scale);
    gl.uniform1f(u("u_warp"), warp);
    gl.uniform1f(u("u_seed"), uSeed);
    gl.drawArrays(gl.TRIANGLES, 0, 3);
  }
}

function linkProgram(gl: WebGL2RenderingContext, vertSrc: string, fragSrc: string): WebGLProgram {
  const prog = gl.createProgram()!;
  for (const [type, src] of [
    [gl.VERTEX_SHADER, vertSrc],
    [gl.FRAGMENT_SHADER, fragSrc],
  ] as const) {
    const sh = gl.createShader(type)!;
    gl.shaderSource(sh, src);
    gl.compileShader(sh);
    if (!gl.getShaderParameter(sh, gl.COMPILE_STATUS)) {
      const log = gl.getShaderInfoLog(sh);
      gl.deleteShader(sh);
      throw new Error(`shader compile failed: ${log}`);
    }
    gl.attachShader(prog, sh);
    gl.deleteShader(sh);
  }
  gl.linkProgram(prog);
  if (!gl.getProgramParameter(prog, gl.LINK_STATUS)) {
    const log = gl.getProgramInfoLog(prog);
    gl.deleteProgram(prog);
    throw new Error(`program link failed: ${log}`);
  }
  return prog;
}

registerBackend("gpu", () => new GpuBackend());
