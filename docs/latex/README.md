# Compilación del informe LaTeX

## Requisitos

- Distribución LaTeX: **MiKTeX**, **TeX Live** o [Overleaf](https://www.overleaf.com)
- Compilar desde la carpeta `docs/`

## Compilar localmente (PowerShell)

```powershell
cd docs
pdflatex -interaction=nonstopmode informe-tecnico-final.tex
pdflatex -interaction=nonstopmode informe-tecnico-final.tex
```

Ejecutar **dos veces** para actualizar índice, lista de figuras y referencias cruzadas.

## Estructura

```text
docs/
├── informe-tecnico-final.tex    # Documento principal
├── latex/
│   ├── preamble.tex             # Paquetes y estilos
│   ├── capitulo01.tex … capitulo08.tex
│   └── referencias.tex
└── diagrams/                    # Figuras PNG
    ├── 01-arquitectura.png
    ├── 02-flujo-insercion.png
    ├── 03-flujo-datos.png
    └── 04-benchmark.png
```

## Errores corregidos (junio 2026)

- Definición del lenguaje **Go** para `listings` (no existe por defecto).
- Color **NavyBlue** habilitado con `dvipsnames` en `xcolor`.
- Comando `\go{...}` para código inline con genéricos, llaves y `<`.
- Captions de listings sin fórmulas en argumentos opcionales.
- Estilo `pscode` para variables PowerShell con `$`.
- Orden correcto de paquetes (`fancyhdr`, `titlesec` antes de usarlos; `hyperref` al final).
- Gráfico pgfplots con escala logarítmica estable.
- `\phantomsection` en capítulos sin numeración para enlaces del índice.


- Portada académica con metadatos en tabla
- Resumen y palabras clave
- Índice general, lista de figuras y tablas
- 8 capítulos con ecuaciones numeradas
- Diagramas TikZ + imágenes PNG
- Código con `listings` (Go, SQL, JSON)
- Tablas con `booktabs`
- Referencias y anexos

## Overleaf

1. Subir la carpeta `docs/` completa (manteniendo estructura).
2. Establecer `informe-tecnico-final.tex` como documento principal.
3. Compilar con pdfLaTeX.
