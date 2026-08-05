@echo off
set PROJECT_ROOT=%~dp0
set PATH=%PROJECT_ROOT%;%PATH%
go test %*