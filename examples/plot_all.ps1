Push-Location (Join-Path $PSScriptRoot '..')
try {
.\.venv\Scripts\python.exe plot_pit.py cmd/mineflow/bauxitemed_output.txt 120 120 26 20 20 20 cmd/mineflow/bauxitemed_output.png
.\.venv\Scripts\python.exe plot_pit.py cmd/mineflow/casestudy_1827500_output.txt 170 215 50 3.5 3.5 3.5 cmd/mineflow/casestudy_1827500_output.png
.\.venv\Scripts\python.exe plot_pit.py cmd/mineflow/cucase_output.txt 170 215 50 3.5 3.5 3.5 cmd/mineflow/cucase_output.png
.\.venv\Scripts\python.exe plot_pit.py cmd/mineflow/cupipe_output.txt 180 180 85 10 10 10 cmd/mineflow/cupipe_output.png
.\.venv\Scripts\python.exe plot_pit.py cmd/mineflow/cusim_2754000_output.txt 180 180 85 10 10 10 cmd/mineflow/cusim_2754000_output.png
.\.venv\Scripts\python.exe plot_pit.py cmd/mineflow/mclaughlingeo_output.txt 140 296 68 25 25 20 cmd/mineflow/mclaughlingeo_output.png
.\.venv\Scripts\python.exe plot_pit.py cmd/mineflow/sim2d76_output.txt 75 1 40 1 1 1 cmd/mineflow/sim2d76_output.png
} finally {
Pop-Location
}
