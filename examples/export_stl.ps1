Push-Location (Join-Path $PSScriptRoot '..')
try {
python export_pit_stl.py cmd/mineflow/cupipe_output.txt 180 180 85 10 10 10 45
python export_pit_stl.py cmd/mineflow/mclaughlingeo_output.txt 140 296 68 25 25 20 45 
python export_pit_stl.py cmd/mineflow/bauxitemed_output.txt 120 120 26 20 20 20 45 
python export_pit_stl.py cmd/mineflow/casestudy_1827500_output.txt 170 215 50 3.5 3.5 3.5 45
} finally {
Pop-Location
}
