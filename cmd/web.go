package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// webHTML 是一段原生零依赖的 HTML+JS 前端代码。
const webHTML = `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <title>IPATool 图形化面板</title>
    <style>
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; background: #f0f2f5; margin: 0; padding: 20px; color: #333;}
        .container { max-width: 750px; margin: 0 auto; background: white; padding: 30px; border-radius: 10px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); }
        h1 { font-size: 26px; color: #0071e3; margin-top: 0;}
        .section { margin-bottom: 25px; padding-bottom: 20px; border-bottom: 1px solid #eaeaea; }
        .section h3 { margin-top: 0; color: #444; }
        input { width: 100%; padding: 12px; margin: 5px 0 15px 0; border: 1px solid #ccc; border-radius: 6px; box-sizing: border-box; font-size: 14px;}
        button { width: 100%; padding: 14px; background: #0071e3; color: white; border: none; border-radius: 6px; font-size: 16px; font-weight: bold; cursor: pointer; transition: 0.2s; }
        button:hover { background: #005bb5; }
        pre { background: #1e1e1e; color: #569cd6; padding: 15px; border-radius: 6px; overflow-x: auto; white-space: pre-wrap; font-size: 14px; line-height: 1.5; min-height: 150px; margin-top: 10px;}
        .note { font-size: 12px; color: #666; margin-top: -10px; margin-bottom: 15px; display: block; }
    </style>
</head>
<body>
    <div class="container">
        <h1>📦 IPATool 图形化操作台</h1>
        
        <div class="section">
            <h3>1. 登录 Apple ID (必做)</h3>
            <input type="text" id="email" placeholder="输入苹果账号 (邮箱)">
            <input type="password" id="password" placeholder="输入账号密码">
            <!-- [新增] 验证码输入框 -->
            <input type="text" id="authCode" placeholder="【若提示需要双重认证】请填入手机收到的 6 位验证码 (首次登录请留空)">
            <button onclick="login()">登录 Apple ID</button>
        </div>

        <div class="section">
            <h3>2. 搜索 App 获取 Bundle ID</h3>
            <input type="text" id="keyword" placeholder="搜索关键字，例如: runwayml" value="runwayml">
            <button onclick="runCmd(['search', v('keyword')])">🔍 在 App Store 搜索应用</button>
        </div>

        <div class="section">
            <h3>3. 查询历史版本数字 ID (External Version ID)</h3>
            <span class="note">查询结果是一串数字ID，建议去第三方网站对照找出你需要版本(如 82.0.5)对应的数字。</span>
            <input type="text" id="bundleId" placeholder="App 包名 (例如: com.runwayml.ios)" value="com.runwayml.ios">
            <button onclick="runCmd(['list-versions', '-b', v('bundleId')])">📋 查询所有历史版本 ID</button>
        </div>

        <div class="section">
            <h3>4. 一键下载</h3>
            <input type="text" id="versionId" placeholder="粘贴你要下载的具体历史版本数字 ID (必填)">
            <input type="text" id="savePath" placeholder="保存位置和文件名 (例如: D:\runwayml.ipa)">
            <button onclick="runCmd(['download', '--purchase', '-b', v('bundleId'), '--external-version-id', v('versionId'), '-o', v('savePath')])">🚀 开始下载</button>
        </div>

        <h3>运行日志：</h3>
        <pre id="output">准备就绪，请按顺序操作...</pre>
    </div>

    <script>
        function v(id) { return document.getElementById(id).value; }
        
        // 专为登录定制的逻辑
        function login() {
            let args = ['auth', 'login', '--email', v('email'), '--password', v('password')];
            let code = v('authCode');
            // 如果用户填了验证码，就带上 --auth-code 参数
            if (code) {
                args.push('--auth-code', code);
            }
            runCmd(args);
        }

        async function runCmd(argsArray) {
            const out = document.getElementById('output');
            out.innerText = "【执行中，请耐心等待，请勿刷新页面】...\n(如涉及网络下载或获取验证码，可能需要几十秒)\n----------------------------------------\n";
            
            try {
                // 改用 JSON 数组传参，完美解决密码中有空格导致断裂的问题
                const response = await fetch("/run", { 
                    method: "POST", 
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(argsArray) 
                });
                let text = await response.text();
                // [新增] 剔除终端产生的 ANSI 颜色控制字符，让日志更清爽
                text = text.replace(/\x1B\[[0-9;]*[a-zA-Z]/g, '');
                out.innerText += text;
            } catch (e) {
                out.innerText += "\n[网络请求失败]：" + e;
            }
        }
    </script>
</body>
</html>
`

// webCmd 负责构建一个新的 "web" 终端子命令，启动本地可视化网页
func webCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "web",
		Short: "启动图形化 Web 控制面板",
		RunE: func(cmd *cobra.Command, args []string) error {
			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = w.Write([]byte(webHTML))
			})

			http.HandleFunc("/run", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost { return }
				
				var cmdArgs []string
				// 接收前端传来的 JSON 数组参数
				if err := json.NewDecoder(r.Body).Decode(&cmdArgs); err != nil || len(cmdArgs) == 0 {
					w.Write([]byte("❌ 参数解析失败"))
					return
				}

				exePath, err := os.Executable()
				if err != nil {
					exePath = "ipatool" 
				}

				execCmd := exec.Command(exePath, cmdArgs...)
				// 强制输出纯文本，无视交互等待
				execCmd.Args = append(execCmd.Args, "--format", "text", "--non-interactive")

				out, err := execCmd.CombinedOutput()
				if err != nil {
					w.Write([]byte(fmt.Sprintf("❌ 执行异常: %v\n\n", err)))
				}
				w.Write(out)
			})

			dependencies.Logger.Log().Msg("🎉 可视化面板已启动！请打开浏览器访问: http://127.0.0.1:8080")
			return http.ListenAndServe(":8080", nil)
		},
	}
	
	return cmd
}
