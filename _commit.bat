@echo off
cd /d E:\SynologyDrive\5_dm_cli
git add -A
git commit -F _commit_msg.txt
del _commit_msg.txt
del _commit.bat
