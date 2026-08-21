Write-Output "bauxitemed.dat"
./mineflow --regular 120 120 26 45 ../../data/bauxitemed.dat bauxitemed_output.txt
Write-Host "=================================" -ForegroundColor Red

Write-Output "casestudy_1827500.dat"
./mineflow --regular 170 215 50 45 ../../data/casestudy_1827500.dat casestudy_1827500_output.txt
Write-Host "=================================" -ForegroundColor Red

Write-Output "cusim_2754000.dat"
./mineflow --regular 180 180 85 45 ../../data/cusim_2754000.dat cusim_2754000_output.txt
Write-Host "=================================" -ForegroundColor Red

Write-Output "sim2d76.dat"
./mineflow --regular 75 1 40 45 ../../data/sim2d76.dat sim2d76_output.txt
Write-Host "=================================" -ForegroundColor Red

Write-Output "mclaughlingeo.dat"
./mineflow --regular 140 296 68 45 ../../data/mclaughlingeo.dat mclaughlingeo_output.txt
Write-Host "=================================" -ForegroundColor Red

Write-Output "cupipe.dat"
./mineflow --regular 180 180 85 45 ../../data/cupipe.dat cupipe_output.txt
Write-Host "=================================" -ForegroundColor Red 

Write-Output "cucase.dat"
./mineflow --regular 170 215 50 45 ../../data/cucase.dat cucase_output.txt

Write-Output "============FINISHED============"