@echo off
echo 🚀 Setting up Git repository...

REM Initialize git if not exists
if not exist .git (
    echo 📁 Initializing Git repository...
    git init
)

REM Add all files
echo 📋 Adding files to Git...
git add .

REM Commit changes
echo 💾 Committing changes...
git commit -m "Initial commit: News Aggregator API microservices system"

REM Add remote origin
echo 🔗 Adding remote origin...
git remote add origin https://github.com/hungprovip123/News-Aggregator-API.git

REM Set main branch
echo 🌿 Setting main branch...
git branch -M main

REM Push to GitHub
echo 🚀 Pushing to GitHub...
git push -u origin main

echo ✅ Successfully pushed to GitHub!
pause 