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

### 2. Estado na URL para os painéis de gradiente / noisefield / paleta

`CLAUDE.md` (secção *Determinism*): *"todo estado da ferramenta … deve ser URL-encodável
para um resultado ser compartilhável e reproduzível"*.

- Os 6 filtros fazem isto: `web/src/state.ts` (`?fx=<nome>&p.<chave>=<valor>`,
  `readStateFromURL` / `writeStateToURL`).
- Os painéis novos (`generators-panel.ts`, `palette-panel.ts`) **não** — as structs `Params`
  em Go são serializáveis (JSON), mas o front nunca as escreve/lê na URL.
- Impacto maior no **gradiente generativo**: foi desenhado com `Seed` + `Time` explícitos
  justamente para ser reproduzível a partir de um link; hoje não é.

**O que falta:** estender `state.ts` (ou um módulo paralelo) para serializar o modo ativo
(filtro | gradiente multi-stop | gradiente generativo | paleta) e os params do painel
correspondente. Formato sugerido para stops/spots:
`stops=hex@pos-hex@pos&type&angle&interp&hue&ease` (ver `gurade-spec.md`).

---

## Polimento menor

| Item | Estado atual | Nota |
|---|---|---|
| Handles de spot arrastáveis sobre o canvas (Mesh/Freeform/Flow) | x/y numéricos no painel | É a interação-assinatura do toy de referência (anéis arrastáveis) |
| Trocar o motor deixa o Worker antigo vivo | `resetBackend()` existe mas não é chamado; sem `dispose()` no `FilterBackend` | Vazamento pequeno; só ao alternar motor repetidamente |
| Exportar a paleta inteira (CSS custom properties / array JSON / `.ase`) | swatch copia 1 hex no clique (`web/src/ui/widgets.ts` `swatchStrip`) | — |
| `Control.showIf` em outros filtros | só aplicado ao `dither` | `glitch`: params de faixa só fazem sentido com `sliceCount > 0`; `ascii`: `background` só com `colored` |
| Split "preview pequeno / export grande" nos painéis de gerador | renderiza no tamanho escolhido direto (limitado a 1024²) | Filtros têm isto via `web/src/preview.ts` + `settings.previewMaxDim` |
| Divergência GPU vs CPU no noisefield orgânico | ~30–45 de média nos campos Mesh/Freeform/Flow (`float` highp vs `float64`) | Documentado; port CPU é autoritativo para export. Geométricos batem ao pixel |

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
