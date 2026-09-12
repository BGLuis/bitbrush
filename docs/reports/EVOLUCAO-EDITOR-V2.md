# Relatório Técnico — Evolução Arquitetural BitBrush v2

**Modo:** A — Análise e proposta (trabalho não implementado)
**Data:** 2026-09-12
**Escopo:** Pipeline de filtros encadeável · Editor de fotos completo · GIF avançado · Recorte e remoção de fundo

---

## 1. Contexto e ponto de partida

O BitBrush é uma aplicação web 100% estática para geração algorítmica de imagens.
Sua arquitetura é dividida em três camadas estritamente separadas
(evidência: `CLAUDE.md` §Architecture, linhas 74–86):

1. **Core Go puro** — `internal/` (sem `syscall/js`, testável com `go test`)
2. **Adaptador WASM** — `cmd/wasm/` (único arquivo que importa `syscall/js`)
3. **Shell Svelte 5** — `web/src/` (sem matemática de pixels)

O estado atual, verificado em `2026-09-12`, é:

| Indicador | Valor medido |
|---|---|
| Pacotes `internal/` com testes | 18/18 passando (`go test -count=1 ./internal/...`) |
| Filtros registrados em `internal/filters/registry.go` | 26 arquivos com `Register(` |
| Geradores no registry de `internal/generators/` | 7 pacotes (truchet, harmonograph, attractor, contours, flowfield, lsystem, flame) |
| Tamanho de `main.wasm` (Go padrão, `-trimpath -ldflags="-s -w"`) | **5,7 MB** / **1,6 MB** gzip |
| Dependências externas | `golang.org/x/image v0.45.0` (única) |
| Testes em `internal/compositor/` | 22 funções `Test*` nos 4 arquivos `_test.go` |

Não há `docs/reports/` pré-existente: esta é a primeira entrada.

---

## 2. O que já existe e é reutilizável

As quatro frentes de evolução não partem do zero.
A tabela abaixo mapeia o que foi lido no repositório e o que cada frente aproveita diretamente.

### 2.1 Pipeline de filtros

O loop de pipeline **já existe** em `internal/compositor/evaluate.go` L93–L99:

```go
cur := src
for j, st := range layer.Chain {
    cur, err = filters.Apply(st.Filter, cur, st.Params)
    if err != nil {
        return nil, fmt.Errorf("compositor: layer %d stage %d (%q): %w", i, j, st.Filter, err)
    }
}
```

O tipo `Stage{Filter string; Params filters.Params}` também já existe na linha 30 do mesmo arquivo.
O que **não existe** é uma função pública que exponha esse loop fora do compositor — hoje ele
só é acessível via `Evaluate(spec, base, extras)` com uma `Spec` completa.

O `FilterBackend` em `web/src/backend.ts` tem 11 métodos (linhas 33–90).
O método `applyPipeline` não está declarado — qualquer chamada exige adicioná-lo à interface
e às três implementações (`cpu.ts`, `worker.ts`, `gpu.ts`).

### 2.2 Editor de fotos

O `internal/compositor` já suporta cinco `SourceKind` (linhas 21–27 de `evaluate.go`):
`base`, `image`, `generator`, `gradient`, `noisefield`.
A struct `Layer` já carrega `Chain []Stage`, `Blend BlendMode`, `Opacity float64` e `Fit FitMode`.

Os 14 blend modes separáveis estão implementados em `internal/compositor/blend.go` (linhas 10–25).
Os quatro modos não-separáveis (`hue`, `saturation`, `color`, `luminosity`) estão **ausentes** —
mencionados no BACKLOG.md como trabalho adiado.

A dependência `golang.org/x/image v0.45.0` (confirmada em `go.mod`) já inclui
`golang.org/x/image/font`, `golang.org/x/image/font/basicfont`, `golang.org/x/image/font/opentype`
e `golang.org/x/image/font/gofont` — verificado via `go list -m -f '{{.Dir}}' golang.org/x/image`.
Isso significa que **renderização de texto no core Go não precisa de nova dependência**.

### 2.3 GIF avançado

`internal/anim/noisefield.go` já implementa `RenderNoiseField(start, end noisefield.Params,
opt Options, w, h int) ([]byte, error)` (linhas 44–50) — o GIF do noisefield **já existe no core Go**.

O global WASM `bitbrushRenderNoiseFieldGIF` e o método `renderNoiseFieldGIF` no `FilterBackend`
também já existem (`web/src/backend.ts` linhas 61–67; `web/src/backends/worker.ts` linhas 145–161).

O que **não existe** é a exposição desse caminho na UI: nenhum botão ou painel liga ao
`renderNoiseFieldGIF` do backend.

`internal/anim/gif.go` usa palette única compartilhada (median-cut via `internal/palette`,
50 000 amostras, `EncodeGIF` linhas 30–83). Não há dithering configurável, paleta por frame,
ou controle de `LoopCount` distinto de "forever/once".

### 2.4 Recorte e remoção de fundo

Não existe nenhum pacote `internal/crop` nem análogo.
Não existe filtro de remoção de fundo no registry de `internal/filters/`.
O BACKLOG.md menciona "overlay de texto" e "presets de aspect-ratio" como **fora do escopo**
do produto original — recorte não é citado, mas tampouco foi implementado.

---

## 3. Análise por frente

### Frente 1 — Pipeline de filtros encadeável

**Lacuna real:** função pública `filters.Pipeline(img, []Stage) (*image.RGBA, error)` não existe.
O loop equivalente em `evaluate.go` L93–L99 tem 7 linhas e zero dependências circulares.
Extraí-lo para `internal/filters/pipeline.go` é uma refatoração de baixo risco.

**Impacto na fronteira WASM:**
O global `bitbrushApplyFilter` (`cmd/wasm/main.go` L46–L72) recebe nome + params para um único filtro.
Um novo global `bitbrushApplyPipeline(rgba, w, h, chainJSON)` segue o mesmo padrão:
`decodeImage` → `filters.Pipeline` → `CopyBytesToJS`.
O `chainJSON` é um array `[{filter, params}, ...]` — mesma estrutura de `Layer.Chain`
que já cruza a fronteira no compositor.

**Impacto no shell:**
O `FilterBackend` precisa de um método `applyPipeline`. As três implementações
(`cpu.ts`, `worker.ts`, `gpu.ts`) precisam implementá-lo.
O worker precisa de um `op: "pipeline"` no `filter-worker.ts`.
A UI precisa de `PipelineEditor.svelte` com drag-and-drop de estágios e `ParamForm.svelte`
por estágio (componente já existe em `web/src/app/ParamForm.svelte`).

**Estimativa de esforço (⚠️ estimado, não medido):**
- Core Go: ~40–60 linhas (extração do loop + nova assinatura pública)
- Adapter WASM: ~25–35 linhas (novo global + decode/encode)
- Backend TS (3 implementações + worker): ~80–120 linhas
- UI Svelte (PipelineEditor): ~150–250 linhas

**Riscos:**
Nenhum breaking change nas assinaturas existentes.
O `ioBuf` compartilhado em `cmd/wasm/main.go` L44 é seguro: chamadas são serializadas
(único thread WASM).

---

### Frente 2 — Editor de fotos completo

#### 2a. Camada de texto

**O que falta:** novo pacote `internal/text` + `SourceText SourceKind` no compositor.

`golang.org/x/image/font` expõe `font.Drawer` para rasterizar glifos em `*image.RGBA` —
confirmado pela saída de `go doc golang.org/x/image/font`.
O pacote `golang.org/x/image/font/gofont` inclui uma família de fontes Go embutidas
(sans-serif, mono) sem dependência de sistema.
`golang.org/x/image/font/opentype` permite carregar `.ttf`/`.otf` via `//go:embed`.

**Integração no compositor:**
`resolveSource` em `evaluate.go` L109–L142 recebe um novo `case SourceText:` que chama
`text.Render(params, w, h)`.
A struct `Layer` não precisa de novos campos: `GenParams json.RawMessage` já carrega
parâmetros genéricos.

**Riscos:**
- Impacto no tamanho do WASM ao embutir fonte: ⚠️ **não medido** — requer build de prova.
- Compatibilidade com TinyGo 0.42.0: `opentype` usa reflexão; ⚠️ **não testado**.
- Rasterização em Go puro é pixel-a-pixel; para preview, usar resolução reduzida
  (já feito pelo `settings.previewMaxDim` existente).

#### 2b. Máscaras e efeitos por região

**O que falta:** campo `Mask *Mask` em `Stage` + `internal/mask` com tipos `rect`, `ellipse`, `luma`.

`Stage` atualmente (`evaluate.go` linhas 30–33):

```go
type Stage struct {
    Filter string         `json:"filter"`
    Params filters.Params `json:"params"`
}
```

Adicionar `Mask *Mask` com `omitempty` é uma mudança aditiva: receitas existentes
sem máscara continuam funcionando.

A lógica em `evaluate.go` L94–L99 muda de:
```go
cur, err = filters.Apply(st.Filter, cur, st.Params)
```
para:
```go
filtered, err = filters.Apply(st.Filter, cur, st.Params)
cur = mask.Compose(cur, filtered, st.Mask)  // se Mask != nil
```

**Tipos de máscara v1 recomendados:** `rect`, `ellipse` (geometria paramétrica) e `luma`
(threshold de luminância). `brush` (pincel livre) exige serialização de coordenadas de pontos —
pode exceder limite de URL; melhor deixar para v2 ou armazenar em `localStorage`.

#### 2c. Anotações e desenho

**O que falta:** novo `SourceDraw SourceKind` + `internal/draw` (strokes bezier).

Strokes são segmentos Bézier cúbicos normalizados em `[0,1]²` — determinísticos, serialização compacta.
A rasterização usa `golang.org/x/image/vector`.

**Riscos:**
- `golang.org/x/image/vector`: disponibilidade na v0.45.0 ⚠️ **não verificada** —
  confirmar antes de assumir.
- Strokes com muitos pontos crescem o `cs=`: ⚠️ **não medido** — necessita análise de
  serialização antes de implementar.

#### 2d. Transformação de camadas de imagem

`Layer` não tem campo de transformação (posição, escala, rotação).
`Fit FitMode` cobre apenas `cover`, `contain`, `fill`, `none`.

Para posicionamento livre, adicionar `Transform *Transform` à `Layer`:

```go
type Transform struct {
    OffsetX   float64 `json:"offsetX,omitempty"`
    OffsetY   float64 `json:"offsetY,omitempty"`
    Scale     float64 `json:"scale,omitempty"`
    RotateDeg float64 `json:"rotateDeg,omitempty"`
}
```

Rotação de imagem usa `golang.org/x/image/draw` — já é dependência direta de
`internal/anim/render.go` L8 (confirmado).

---

### Frente 3 — GIF com controles avançados

#### 3a. GIF do noisefield (na UI)

**Status real:** `anim.RenderNoiseField` e o método `renderNoiseFieldGIF` do `FilterBackend`
**já existem** e estão conectados no worker (`worker.ts` linhas 145–161).
O que falta é **100% de UI**: nenhum botão no painel do gerador chama `backend.renderNoiseFieldGIF`.

Custo: adicionar o botão "Exportar GIF" ao `GeneratorPanel.svelte` e conectar ao
`backend.renderNoiseFieldGIF` — apenas shell, zero Go.

#### 3b. GIF da pilha Compose

**O que falta:** `internal/anim/compose.go` (novo) + global WASM.

O compositor já é determinístico: `Evaluate(spec, base, extras)` com a mesma entrada
produz saída idêntica. Interpolar entre duas `Spec` requer identificar quais campos
são numéricos por camada — a mesma lógica de `Interpolate` de `anim.go` L132–L167,
aplicada a `json.RawMessage` de cada `GenParams`.

**Alternativa mais simples para v1:** interpolar apenas os `GenParams` numéricos de cada
camada, sem adicionar/remover camadas entre keyframes. Restrição documentável na UI.

#### 3c. Controles avançados de GIF

`anim.Options` atual (`anim.go` L26–L31):

```go
type Options struct {
    Frames       int
    FPS          int
    LoopForever  bool
    PingPong     bool
    MaxDimension int
}
```

Campos a adicionar para os novos controles:

| Controle novo | Campo Go novo | Observação |
|---|---|---|
| Loop count (1x/2x/∞) | `LoopCount int` | substituir `LoopForever bool` ou coexistir |
| Easing entre keyframes | `Easing string` por `Keyframe` | ex: "ease-in", "cubic-bezier(…)" |
| Dithering configurável | `Dither string` em `GIFOptions` | encoder atual usa OKLab mapper, não `draw.FloydSteinberg` |
| Paleta por frame | `PerFramePalette bool` | aumenta tamanho do GIF |
| Paleta customizada | `CustomPalette []string` | alimentada por `internal/palette` |

⚠️ O encoder atual (`gif.go` L56) usa `palette.NewMapper(pal, palette.SpaceOKLab)` — não usa
`draw.FloydSteinberg`. Dithering configurável requer mudança em `paletteFrame`.

#### 3d. Easing entre keyframes

`Interpolate` (`anim.go` L132–L167) chama `phaseAt` para obter `t` em `[0,1]`.
Inserir easing requer: (1) `Keyframe` ganha campo `Easing string`; (2) o passo de
interpolação aplica a curva ao `local` em L160–L163.

A função `cubicBezier(x1, y1, x2, y2 float64) func(float64) float64` é ~30 linhas em Go
(aproximação numérica da curva paramétrica CSS `cubic-bezier`).

---

### Frente 4 — Recorte e remoção de fundo

#### 4a. Recorte

Não existe nada em `internal/` para recorte. `image.RGBA.SubImage(r image.Rectangle)` já
existe na stdlib — `internal/crop` resume-se a converter coordenadas normalizadas em
`image.Rectangle` e chamar `SubImage`.

O recorte deve ser **não-destrutivo no estado**: armazenado em `ui.crop: CropRect | null`
na store Svelte e aplicado no render loop de `app/lib/render.ts` como pré-processamento.
Handles de recorte sobre o canvas no `Stage.svelte` (4 bordas ou canto, overlay de preview)
são toda a mudança de UI necessária.

#### 4b. Remoção de fundo algorítmica

**Opção A — Chroma/Luma Key:** implementada como filtro normal registrado no registry.
Assinatura: `func(src *image.RGBA, params Params) (*image.RGBA, error)` — idêntica a todos
os outros filtros. Toca apenas dois arquivos: `internal/filters/bgremove.go` (novo) +
descritor em `web/src/ui/controls.ts`.

**Opção B — Magic Wand (flood fill):** exige coordenada de pixel como entrada.
`Params` é `map[string]any` — `seedX` e `seedY` podem ir como campos numéricos, mas requerem
que o shell envie as coordenadas do clique. Global WASM separado (`bitbrushMagicWand`)
para não poluir `applyFilter`.

**Opção C — ONNX/ML:** fora do escopo v2. A arquitetura de backend seam (`backend.ts`)
suporta adição futura via novo `BackendKind = "ml"` sem mudanças retroativas.

**Impacto no tamanho do WASM:** Opção A e B não adicionam dependências; impacto desprezível
(~5–15 KB de Go compilado estimado).

---

## 4. Dependências entre frentes

```
Frente 1 (pipeline)     ──→  desbloqueia Frente 2b (mask.Compose usa o loop de pipeline)
Frente 4b (bgremove)    ──→  produz canal alpha que alimenta Frente 2b (MaskKind "alpha")
Frente 3a (GIF noisefield UI) ──→  zero Go; pode ser feito imediatamente
Frente 4a (crop)        ──→  completamente independente
Frente 2a (texto)       ──→  independente de todas, mas requer validação de tamanho WASM primeiro
Frente 3b (GIF compose) ──→  requer Frente 2b estável; é a mais complexa
```

Ordem recomendada para minimizar retrabalho:

1. **Frente 3a** — zero risco, zero Go, só UI. Fecha lacuna documentada no BACKLOG.
2. **Frente 1** — extrair `filters.Pipeline`; adicionar `applyPipeline` ao seam.
3. **Frente 4a** — recorte; 100% independente.
4. **Frente 4b opção A** — remoção de fundo como filtro; 2 touch points.
5. **Frente 2a** — camada de texto (build de prova do WASM com fonte embutida primeiro).
6. **Frente 2b** — máscaras (depende do pipeline).
7. **Frente 3c/3d** — controles avançados de GIF + easing.
8. **Frente 3b** — GIF da pilha Compose (mais complexo).
9. **Frente 2c/2d** — anotações e transformações de camada.

---

## 5. Riscos e o que não foi verificado

| Item | Risco | Status |
|---|---|---|
| Impacto no WASM ao embutir fonte | ~200–400 KB estimado; pode exceder orçamento | ⚠️ **Não medido** — requer build de prova |
| `golang.org/x/image/vector` disponível em v0.45.0 | Sub-pacote necessário para strokes | ⚠️ **Não verificado** |
| TinyGo 0.42.0 + `opentype` (reflexão) | Build de produção pode falhar | ⚠️ **Não testado** — requer `tinygo build` de prova |
| Tamanho de `cs=` com strokes | ~50+ pontos pode exceder limite de URL | ⚠️ **Não medido** |
| Dithering configurável | Encoder atual usa OKLab mapper, não `draw.FloydSteinberg` | Confirmado em `gif.go` L56; mudança necessária |
| Blend modes não-separáveis | `hue`/`saturation`/`color`/`luminosity` ausentes em `blend.go` | Confirmado no BACKLOG.md; não bloqueia as quatro frentes |
| Loop sem costura no GIF noisefield | Apenas `isAngle360` verificado; phase closure do fBm não é garantido | Documentado no BACKLOG.md; aceitável para v1 |

---

## 6. Resumo executivo

O repositório está em estado sólido: 18/18 pacotes passando, WASM de 5,7 MB (1,6 MB gzip),
única dependência externa. As quatro frentes são viáveis sem violar os princípios do projeto
(CPU-first, determinismo, 100% estático, registry aberto).

A frente de **maior retorno imediato com menor risco** é a exposição do GIF do noisefield
na UI (Frente 3a): a infraestrutura Go e o contrato de backend já existem — só falta o botão.

A frente de **maior impacto arquitetural** é o pipeline (Frente 1): extrai uma duplicação
real entre o compositor e o modo single-filter, e desbloqueia as máscaras.

A frente de **maior incerteza técnica** é a camada de texto (Frente 2a): o impacto no
tamanho do WASM com fonte embutida e a compatibilidade com TinyGo precisam ser medidos
antes de qualquer comprometimento de design.

Cinco afirmações neste relatório dependem de medição e ainda não foram medidas
(marcadas ⚠️ na seção 5).
