@echo off
set finalname=%1
set finalname=%finalname:"=%
set /A counter=1
set files=
:loop
if exist "%finalname%.%counter%" (
	set files=%files%"%finalname%.%counter%"+
	set /A counter=%counter%+1
	goto loop
)

set files=%files:~0,-1%
rem excute merge command
copy /b %files% "%finalname%"