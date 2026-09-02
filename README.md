<div align="center">

<!-- Badges de Status do GitHub -->
![GitHub Stars](https://www.shieldcn.dev/github/stars/bgluis/bitbrush.svg?variant=secondary&size=sm)
![GitHub Forks](https://www.shieldcn.dev/github/forks/bgluis/bitbrush.svg?variant=secondary&size=sm)
![Watchers](https://www.shieldcn.dev/github/watchers/bgluis/bitbrush.svg?variant=secondary&size=sm)
![Contributors](https://www.shieldcn.dev/github/contributors/bgluis/bitbrush.svg?theme=emerald&size=sm)
![License](https://www.shieldcn.dev/github/license/bgluis/bitbrush.svg?variant=ghost&size=sm)

<br/>

<!-- Badges das Tecnologias Utilizadas -->
![Go](https://www.shieldcn.dev/badge/Go-1.27-00ADD8.svg?logo=go&variant=branded&size=sm)
![TypeScript](https://www.shieldcn.dev/badge/TypeScript-5.7-3178C6.svg?logo=typescript&variant=branded&size=sm)
![Svelte](https://www.shieldcn.dev/badge/Svelte-5-FF3E00.svg?logo=svelte&variant=branded&size=sm)
![WebAssembly](https://www.shieldcn.dev/badge/WebAssembly-wasm-654FF0.svg?logo=webassembly&variant=branded&size=sm)
![Vite](https://www.shieldcn.dev/badge/Vite-6-646CFF.svg?logo=vite&variant=branded&size=sm)

  <h3>BitBrush</h3>
  Estúdio web 100% estático de filtros de imagem, gradientes e paletas algorítmicos — núcleo Go compilado para WebAssembly, shell Svelte 5.

  <br/>

  🇧🇷 **Português** · [🇺🇸 English](./README.en.md)

</div>

# 📖 Sobre

**BitBrush** é uma aplicação web totalmente estática para gerar **filtros de imagem, gradientes e
paletas de cores algorítmicos**. Todo efeito é código determinístico — sem modelos de ML, sem
dados pré-treinados, sem serviços externos de imagem. Essa restrição é um princípio de produto:
limita as dependências e garante que qualquer resultado possa ser reproduzido apenas a partir de
seus parâmetros.

O projeto é dividido em três camadas estritamente separadas:

1. **Núcleo de algoritmos — `internal/`** (Go puro): toda a matemática de filtros, gradientes e
   paletas. Roda e é testado com `go test` em qualquer sistema operacional.
2. **Adaptador WASM — `cmd/wasm/`**: o único ponto que importa `syscall/js`. Registra funções
   nomeadas em `globalThis` e faz o marshalling dos dados na fronteira JS ↔ WASM.
3. **Shell de navegador — `web/src/`** (Svelte 5 + TypeScript + Vite): controles, I/O de canvas
   e estado na URL. Nenhuma matemática de imagem — tudo é delegado ao núcleo Go.

**O que dá para fazer:**

- **Filtros de imagem** — ASCII colorido, pixelate (8-bit), *dithering* por difusão de erro
  (Floyd–Steinberg / Atkinson / Stucki / Jarvis / Sierra / Burkes) e Bayer ordenado, detecção
  de bordas Sobel, *glitch* / RGB shift, quantização de cor (poster), *halftone* (mono / CMYK /
  RGB) e *stippling* de Voronoi.
- **Geradores algorítmicos** — `truchet` (azulejos de Truchet multi-escala), `harmonograph`
  (figura de senoides amortecidas), `attractor` (De Jong / Clifford / Svensson), `contours`
  (mapa topográfico animado), `flowfield` (traços de tinta sumi em campo de fluxo), `lsystem`
  (turtle graphics de L-system) e `flame` (fractal flames / IFS).
- **Gerador de gradientes + ferramenta CSS** — multi-stop, ângulo, interpolação em
  sRGB / linear / HSL / Lab / LCh / OKLab / OKLCh com arcos de matiz do CSS Color 4, *easing*
  entre stops, exportando *hard stops* ou `linear-gradient(… in oklch …)` nativo.
- **Gradiente generativo** — porte determinístico em Go de um *fragment shader*: 8 campos × 8
  estilos × 6 texturas, com `Seed` e `Time` explícitos (anima e é reproduzível).
- **Paletas** — extração a partir de uma imagem (median-cut ou k-means) e geração a partir de
  uma cor-base por regra de harmonia (complementar / análoga / tríade / tétrade /
  split-complementar / monocromática).
- **Exportação de GIF animado** — *keyframe* dos parâmetros de um filtro (início/fim) e
  renderização de N quadros interpolados para um GIF, tudo em CPU.

**Determinismo:** qualquer coisa estocástica recebe um parâmetro inteiro `seed` explícito (sem
`time.Now()`, sem `rand` não-semeado) e todo o estado das ferramentas é codificável na URL — um
resultado é sempre compartilhável e reproduzível.

# 📋 Motivo

O projeto nasceu de uma ideia que surgiu enquanto eu navegava no X (Twitter) e via vários
projetos que geravam padrões geométricos e fractais, além de um gerador de gradientes animado.
Foi criado também como um experimento para o uso de **WebAssembly**, já que eu nunca tinha
mexido com essa tecnologia até então.

# 💻 Como iniciar

### Requisitos

- [Go](https://go.dev/dl/) **1.27+** (exigido pelo `go.mod`; usado no alvo `GOOS=js GOARCH=wasm`)
- [Node.js](https://nodejs.org/) **20 LTS ou superior** (Vite 6)
- [Make](https://www.gnu.org/software/make/) — opcional, mas recomendado: os pontos de entrada
  canônicos estão no `Makefile`
- Um navegador moderno com suporte a **WebAssembly** e **WebGL2**

### Instalação

1. Clone o repositório do projeto:
  ```sh
  git clone https://github.com/BGLuis/bitbrush.git
  ```

2. Navegue até o diretório do projeto:
  ```sh
  cd bitbrush
  ```

3. Instale as dependências do front-end:
  ```sh
  cd web && npm install && cd ..
  ```

4. Suba o ambiente de desenvolvimento (compila o WASM e inicia o Vite):
  ```sh
  make dev
  ```
  Sem o `make`, os passos equivalentes são:
  ```sh
  # 1. compila o núcleo Go para WebAssembly e copia o wasm_exec.js
  GOOS=js GOARCH=wasm go build -o web/public/main.wasm ./cmd/wasm
  cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/public/   # Go <= 1.23: misc/wasm/wasm_exec.js

  # 2. inicia o servidor de desenvolvimento
  cd web && npm run dev
  ```

5. Para gerar o build estático de produção em `web/dist/`:
  ```sh
  make build
  ```

### Comandos úteis

| Comando       | O que faz                                                              |
| ------------- | -------------------------------------------------------------------- |
| `make wasm`   | Compila `cmd/wasm` para `web/public/main.wasm` e copia o `wasm_exec.js` |
| `make dev`    | `make wasm` + `npm run dev` (servidor de desenvolvimento)             |
| `make build`  | `make wasm` + `npm run build` → saída estática em `web/dist/`         |
| `make test`   | `go test ./internal/...`                                             |
| `make check`  | `go vet` + `gofmt -l .` + `svelte-check`                              |
| `make clean`  | Remove artefatos de build e `node_modules`                           |

# 🤝 Contribuidores

 <a href="https://github.com/BGLuis/bitbrush/graphs/contributors">
   <img src="https://contrib.rocks/image?repo=BGLuis/bitbrush"/>
 </a>

# 📄 Licença

Distribuído sob a licença **MIT**. Veja [`LICENSE`](./LICENSE) para mais informações.
