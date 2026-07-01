# AGENTS.md

目标：**直达结果，不制造流程。**

## Git / 推送

- 用户要求提交 / 推送时，必须同时检查 `D:/cocosProject/ai-server` 和 `D:/cocosProject/ai-project` 的 `git status --short --branch`。
- 涉及前后端联动、协议、Classic town/battle/service 的任务，两边有改动就一起提交并推送；只有一边有改动，也要在最终回复说明另一边已检查。
- 推送 GitHub 远端时，默认使用本机 Clash HTTP 代理端口，不改全局 git 配置：`git -c http.proxy=http://127.0.0.1:7890 -c https.proxy=http://127.0.0.1:7890 push -u origin main`。
- 只在本次命令上加 `-c http.proxy=... -c https.proxy=...`；不要写入全局或仓库 git config，除非用户明确要求。
