# BACKLOG — planejado e ainda não implementado

Situação em 2026-09-02. Tudo que estava no escopo v1 (6 filtros, `internal/gradient`,
`internal/noisefield`, `internal/palette`, `internal/anim`) e na camada de infra a–d
(Worker, seam de backend, pipeline de GIF, acelerador GPU) está **feito e verificado**.
O que segue são as pontas soltas.

## Feito depois do v1 (2026-09-02, segunda leva)

- **Filtros novos:** `halftone` (trama AM mono/CMYK/RGB), `stipple` (pontilhado Voronoi
  ponderado, sobre `internal/voronoi`), e `dither` estendido com núcleos de difusão
  (Atkinson/Stucki/Jarvis/Sierra/Burkes) + modo `ordered` (Bayer 2/4/8). Cada um: core +
  registry + descritor em `web/src/ui/controls.ts`. Cobertos por `go test`.
- **Seam de geradores:** `internal/generators` (registry de string, auto-registro via
  `init()`), `internal/genall` (agregador de blank-imports), global
  `bitbrushRenderGenerator(name, paramsJSON, w, h)`, método `renderGenerator` nos três
  backends + op `generator` no worker, `web/src/ui/generators.ts` (descritores).
- **7 geradores:** `truchet`, `harmonograph`, `attractor` (De Jong/Clifford/Svensson),
  `contours` (topográfico animado), `flowfield` (tinta sumi), `lsystem`, `flame` (fractal
  flame). Todos determinísticos por seed, `go test` cobrindo cada um.

**Ponta solta desta leva:** os descritores em `web/src/ui/generators.ts` existem mas **nenhum
painel imperativo os expõe** ainda. É preciso um painel (ou uma seção no
`generators-panel.ts`) que liste os geradores registrados, monte os controles a partir dos
descritores e chame `backend.renderGenerator(...)` — mesmo tipo de fiação que falta ao
noisefield (ver itens 1 e 2 abaixo). Enquanto isso, os geradores só são alcançáveis via a
global WASM diretamente.

---

## Lacunas reais (estavam no plano, não foram feitas)

### 1. Export de GIF animado do gradiente generativo

Interesse explícito desde o início do projeto ("pequenos gifs"). Hoje:

- `internal/anim` só dirige efeitos **registrados em `internal/filters`** (via `filters.Apply`).
- `internal/noisefield` **não** é um filtro registrado — é um gerador com `Params{… Seed, Time}`.
- Resultado: o campo de ruído **anima no preview vivo** (loop rAF em `web/src/generators-panel.ts`,
  e shader GPU em `web/src/backends/gpu.ts`) **mas não exporta GIF**.

**O que falta:**

- Um caminho de timeline para o noisefield: varrer `Time` de 0..duração em N frames →
  `noisefield.Render` por frame → `anim.EncodeGIF`. Pode ser função nova em `internal/anim`
  (ex. `RenderNoiseFieldGIF(params, opt)`) ou pacote irmão.
- Global WASM `bitbrushRenderNoiseFieldGIF(paramsJSON, optionsJSON)` em
  `cmd/wasm/generators.go`, no mesmo formato `{ok, data, error}`.
- Método no `FilterBackend` + op no worker + botão "Exportar GIF" no painel generativo
  (o painel de GIF atual, `web/src/gif.ts`, é acoplado aos filtros).
- Loop sem costura: escolher `Time` final onde a fase do fBm fecha, ou crossfade
  primeiro/último frame.

### 2. Estado na URL para o painel de paleta

`CLAUDE.md` (secção *Determinism*): *"todo estado da ferramenta … deve ser URL-encodável
para um resultado ser compartilhável e reproduzível"*.

- Os 6 filtros e o **gradiente generativo / perceptual** fazem isto: `web/src/state.ts`
  (`?fx=<nome>&p.<chave>=<valor>` para filtros, e
  `?mode=generator&tool=noise&field=...&style=...&cols=...&spots=...` para o gerador).
- Resta apenas o painel de paleta (`palette-panel.ts`) serializar seu estado na URL.

---

## Polimento menor

| Item | Estado atual | Nota |
|---|---|---|
| Handles de spot arrastáveis sobre o canvas (Mesh/Freeform/Flow) | **Feito** (`CanvasSpotsOverlay.svelte`) | Anéis arrastáveis interativos sobre o canvas com badges de nome de cor, hex e suporte a teclado |
| Painel do gerador em Svelte 5 nativo | **Feito** (`GeneratorPanel.svelte`) | Substituiu `generators-panel.ts` legado com 28 presets, bússola, shuffle harmônico e exportação |
| Exportação PNG 1600px e código CSS no gradiente generativo | **Feito** | Render em 1600px e gerador de CSS com aproximação de camadas ou gradientes nativos |
| Trocar o motor deixa o Worker antigo vivo | `resetBackend()` existe mas não é chamado; sem `dispose()` no `FilterBackend` | Vazamento pequeno; só ao alternar motor repetidamente |
| Exportar a paleta inteira (CSS custom properties / array JSON / `.ase`) | swatch copia 1 hex no clique (`web/src/ui/widgets.ts` `swatchStrip`) | — |
| `Control.showIf` em outros filtros | só aplicado ao `dither` | `glitch`: params de faixa só fazem sentido com `sliceCount > 0`; `ascii`: `background` só com `colored` |
| Split "preview pequeno / export grande" nos painéis de gerador | **Feito no gerador** (preview 960px / export 1600px) | Mantém 60fps no preview vivo |
| Divergência GPU vs CPU no noisefield orgânico | ~30–45 de média nos campos Mesh/Freeform/Flow (`float` highp vs `float64`) | Documentado; port CPU é autoritativo para export. Geométricos batem ao pixel |

---

## "Compor" (modo layer stack) — pontas soltas

O núcleo (`internal/compositor`), a global `bitbrushRenderComposite`, o seam de backend, a
store/URL/loop e a UI (`app/compose/`) estão **feitos e verificados** (22 testes em
`go test ./internal/compositor`, preview vivo e hidratação por link testados no navegador).
Adiado:

- **Export GIF de uma pilha.** `Stage.svelte` só monta `<GifPanel>` em `filter`/`generator`.
  Animar a pilha exige `anim.RenderCompose(specStart, specEnd, opts)` + UI de keyframe
  ciente do modo compose.
- **Editores completos de gradiente / ruído por camada.** `LayerEditor` expõe um subconjunto
  (gradiente: 2 cores + espaço + ângulo; ruído: campo/estilo/textura/seed/escala/distorção/
  ângulo + 2 cores). Falta paridade com `GeneratorPanel` (multi-stop, easing, spots) — ou um
  hand-off "editar no modo Gerador e adicionar como camada".
- **Modos de blend não-separáveis** (`hue` / `saturation` / `color` / `luminosity`). v1 traz
  os 14 separáveis; os 4 restantes usam `Lum`/`ClipColor`/`SetLum` do W3C — um braço de
  `switch` cada em `blend.go`.
- **`ui.query` (status bar "Receita") atrasa uma interação** ao entrar em compose/generator —
  `writeComposeStateToURL` usa `debouncedReplaceState` (300ms) e `ui.query` lê `location.search`
  antes do flush. Cosmético; idêntico ao comportamento atual do modo gerador.
- **Bloat de URL** em pilhas profundas: `cs=` já omite chaves default; um `cz=` com
  `CompressionStream` é o próximo passo se necessário.
- **Backend GPU** delega `renderComposite` ao core Go; a store já força worker/CPU quando o
  motor é `gpu`, então `auto` não é afetado.

---

## Adiado por decisão explícita (não são falhas)

- **TinyGo** — otimização de tamanho planejada (`CLAUDE.md` › *Toolchain decisions*):
  *"não está em uso — revisitar quando o tamanho do binário for um problema real"*.
  `web/public/main.wasm` ≈ 5,5 MB (~1,5 MB gzip).
- **Registry de string para gradientes** — `CLAUDE.md` › *Registry pattern* menciona
  "o equivalente para gradientes". Como são 2 pacotes com 1 entrada cada
  (`gradient.Generate` / `noisefield.Render`), ficou sem registry de string.
  Divergência mínima da arquitetura descrita.
- **Recursos do toy de referência fora do escopo BitBrush**: overlay de texto
  (fonte/peso/posição/sombra), presets nomeados, presets de aspect-ratio. Nunca
  estiveram no escopo; ver `gurade-spec.md` para o inventário completo.

---

## Pendência não-código

- **Nada foi commitado ainda.** Tudo `untracked` desde o início (scaffold + PRs 1–6 +
  camada a–d + `internal/{gradient,noisefield,palette,colorspace}` + fiação web + shader GPU).
  Decisão em aberto: fazer o commit inicial agora ou continuar acumulando.
