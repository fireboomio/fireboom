获取 wundergraph 子依赖包：
1）第一次拉代码或没有拉取过分支代码，执行：
git submodule init
git submodule update --remote --recursive
2）然后进入主项目中子项目的目录，执行：
git pull
3) 如果主项目所引用的子项目的 branch 发生了变化，则需要执行：
   git fetch
   git checkout submodule_newbranch
   git pull
4) 更新子项目执行【涵盖第3条目内容】：
   bash ./[pull_submodule.sh](scripts/pull_submodule.sh)

如何和 wundergraph 同步代码？
（1）wundergraphGitSubmodule 目录下：
wundergraphGitSubmodule子目录的分支使用gen分支

1. 配置当前当前 fork 的仓库的原仓库地址
   git remote add upstream <原仓库 github 地址>

2. 查看当前仓库的远程仓库地址和原仓库地址
   git remote -v

3. 获取原仓库的更新。使用 fetch 更新，fetch 后会被存储在一个本地分支 upstream/master 上。
   git fetch upstream

4. 合并到本地分支。切换到本地 master 分支，合并 upstream/master 分支。
   git merge upstream/master

5. 这时候使用 git log 就能看到原仓库的更新了。
   git log

6. 如果需要自己 github 上的 fork 的仓库需要保持同步更新，执行 git push 进行推送
   git push origin main