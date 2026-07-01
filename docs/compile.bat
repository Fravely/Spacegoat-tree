@echo off
REM Compila informe-tecnico-final.tex (dos pasadas para indice y referencias)
cd /d "%~dp0"
where pdflatex >nul 2>&1
if errorlevel 1 (
    echo ERROR: pdflatex no encontrado. Instale MiKTeX o TeX Live, o use Overleaf.
    echo Ver docs\latex\README.md
    exit /b 1
)
echo Compilando informe-tecnico-final.tex ...
pdflatex -interaction=nonstopmode informe-tecnico-final.tex
pdflatex -interaction=nonstopmode informe-tecnico-final.tex
if exist informe-tecnico-final.pdf (
    echo.
    echo Listo: informe-tecnico-final.pdf
) else (
    echo.
    echo La compilacion fallo. Revise informe-tecnico-final.log
    exit /b 1
)
