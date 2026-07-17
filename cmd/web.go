package cmd

import (
	"encoding/json"
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
    <title>IPATool 终极图形化面板</title>
    <style>
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; background: #f0f2f5; margin: 0; padding: 20px; color: #333;}
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 30px; border-radius: 10px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); }
        h1 { font-size: 26px; color: #0071e3; margin-top: 0;}
        .section { margin-bottom: 25px; padding-bottom: 20px; border-bottom: 1px solid #eaeaea; }
        .section h3 { margin-top: 0; color: #444; }
        input { width: 100%; padding: 12px; margin: 5px 0 15px 0; border: 1px solid #ccc; border-radius: 6px; box-sizing: border-box; font-size: 14px;}
        button { width: 100%; padding: 14px; background: #0071e3; color: white; border: none; border-radius: 6px; font-size: 16px; font-weight: bold; cursor: pointer; transition: 0.2s; margin-bottom: 5px;}
        button:hover { background: #005bb5; }
        button:disabled { background: #ccc; cursor: not-allowed; }
        pre { background: #1e1e1e; color: #569cd6; padding: 15px; border-radius: 6px; overflow-x: auto; white-space: pre-wrap; font-size: 14px; line-height: 1.5; min-height: 150px; margin-top: 10px;}
        .note { font-size: 12px; color: #666; margin-top: -10px; margin-bottom: 15px; display: block; }
        .card { border: 1px solid #ddd; padding: 15px; margin-bottom: 10px; border-radius: 8px; background: #fafafa;}
    </style>
</head>
<body>
    <div class="container">
        <h1>📦 IPATool 终极图形化操作台</h1>
        
        <div class="section">
            <h3>1. 登录 Apple ID (必做)</h3>
            <input type="text" id="email" placeholder="输入苹果账号 (邮箱)">
            <input type="password" id="password" placeholder="输入账号密码">
            <input type="text" id="authCode" placeholder="【若提示需要双重认证】请填入手机收到的 6 位验证码 (首次登录请留空)">
            <button onclick="login()">🔐 登录 Apple ID</button>
        </div>

        <div class="section">
            <h3>2. 搜索 App 获取真实包名 (Bundle ID)</h3>
            <input type="text" id="keyword" placeholder="搜索关键字，例如: runwayml" value="runwayml">
            <button onclick="searchApp()">🔍 智能搜索</button>
            <div id="searchResults" style="margin-top: 10px;"></div>
        </div>

        <div class="section">
            <h3>3. 智能历史版本解析器</h3>
            <span class="note">我们将从苹果服务器拉取所有内部数字 ID，并允许你一键解析出它们对应的真实版本号（如 82.0.5）。</span>
            <input type="text" id="bundleId" placeholder="在此填入 App 包名 (从第2步获取)">
            <button onclick="fetchVersions()">📋 拉取所有历史版本 ID 列表</button>
            <div id="versionContainer" style="max-height: 400px; overflow-y: auto; border: 1px solid #ccc; padding: 10px; border-radius:6px; display: none; background:#fff; margin-top:10px;"></div>
        </div>

        <div class="section">
            <h3>4. 一键下载</h3>
            <input type="text" id="versionId" placeholder="目标版本的数字 ID (请在第 3 步列表中点击选定，系统会自动填入此框)">
            <input type="text" id="savePath" placeholder="保存位置和文件名 (例如: D:\RunwayML_历史版.ipa)">
            <button style="background: #28a745;" onclick="downloadApp()">🚀 获取授权并开始下载！</button>
        </div>

        <h3>运行日志：</h3>
        <pre id="output">准备就绪，请按顺序操作...</pre>
    </div>

    <script>
        function v(id) { return document.getElementById(id).value; }
        function logOut(msg) { document.getElementById('output').innerText = msg; }
        function appendLog(msg) { document.getElementById('output').innerText += msg; }

        function extractJSON(text) {
            let res = {};
            let lines = text.split('\n');
            for (let line of lines) {
                if(line.trim() === '') continue;
                try { let obj = JSON.parse(line); Object.assign(res, obj); } catch(e) {}
            }
            return res;
        }

        async function runBackend(argsArray) {
            const response = await fetch("/run", { 
                method: "POST", 
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(argsArray) 
            });
            let text = await response.text();
            text = text.replace(/\x1B\[[0-9;]*[a-zA-Z]/g, ''); 
            return text;
        }

        async function login() {
            logOut("【登录中，请耐心等待】...\n----------------------------------------\n");
            let args = ['auth', 'login', '--email', v('email'), '--password', v('password')];
            if (v('authCode')) args.push('--auth-code', v('authCode'));
            
            let text = await runBackend(args);
            appendLog(text);
            
            if (text.indexOf("2FA code is required") !== -1 || text.indexOf("auth-code") !== -1) {
                alert("⚠️ 触发双重认证！\n请查看手机验证码，填入第一个板块的第三个输入框，然后【再次点击登录】！");
            }
        }

        async function searchApp() {
            logOut("【搜索中，正在联系 App Store】...\n");
            let args = ['search', v('keyword'), '--limit', '3', '--format', 'json'];
            let text = await runBackend(args);
            let data = extractJSON(text);
            
            if (data.apps && data.apps.length > 0) {
                logOut("搜索成功！请在下方选择你的目标 App。\n\n" + text);
                let html = '<h4>搜索结果 (自动排版)：</h4>';
                data.apps.forEach(function(app) {
                    html += '<div class="card">' +
                            '<strong style="font-size:18px;">' + app.name + '</strong> <span style="color:#666;">(最新版本: ' + app.version + ')</span><br>' +
                            '<span style="font-size:13px; color:#e0245e; font-weight:bold;">包名 (Bundle ID): ' + app.bundleID + '</span><br>' +
                            '<button style="margin-top:10px; width:auto; padding:8px 15px; font-size:14px;" onclick="document.getElementById(\'bundleId\').value=\'' + app.bundleID + '\'; alert(\'包名已自动填入第3步！\')">👇 选定此 App</button>' +
                            '</div>';
                });
                document.getElementById('searchResults').innerHTML = html;
            } else {
                logOut("未找到该应用，请检查关键字。\n日志：\n" + text);
            }
        }

        async function fetchVersions() {
            let bundleId = v('bundleId');
            if (!bundleId) return alert("请先填写或搜索选定 Bundle ID");
            
            logOut("【正在拉取历史版本数据库，请稍候】...\n");
            let args = ['list-versions', '-b', bundleId, '--format', 'json'];
            let text = await runBackend(args);
            let data = extractJSON(text);
            
            if (data.externalVersionIdentifiers) {
                logOut("获取成功！请在下方列表中点击解析具体版本号。\n");
                let container = document.getElementById('versionContainer');
                container.style.display = 'block';
                container.innerHTML = '';
                
                let ids = data.externalVersionIdentifiers.reverse();
                ids.forEach(function(id) {
                    let div = document.createElement('div');
                    div.style.padding = '8px';
                    div.style.borderBottom = '1px dashed #eee';
                    div.innerHTML = '<span style="display:inline-block; width:150px; color:#555;">内部 ID: ' + id + '</span>' +
                                    '<button style="width:auto; padding: 6px 15px; font-size:13px; background:#6c757d;" onclick="resolveVersion(\'' + bundleId + '\', \'' + id + '\', this)">🔍 点击解析这是几号版本</button>';
                    container.appendChild(div);
                });
            } else {
                logOut("获取历史版本失败，可能是未购买或包名错误。\n\n" + text);
            }
        }

        async function resolveVersion(bundleId, versionId, btn) {
            btn.innerText = "⏳ 正在从云端读取 Info.plist...";
            btn.disabled = true;
            let args = ['get-version-metadata', '-b', bundleId, '--external-version-id', versionId, '--format', 'json'];
            
            let text = await runBackend(args);
            let data = extractJSON(text);
            
            if (data.displayVersion) {
                let dateStr = new Date(data.releaseDate).toLocaleDateString();
                btn.parentElement.innerHTML = '<span style="display:inline-block; width:150px; color:#555;">内部 ID: ' + versionId + '</span>' +
                                              '<span style="display:inline-block; width:220px; color: #d93025; font-weight:bold; font-size:18px;">版本号: ' + data.displayVersion + '</span>' +
                                              '<span style="display:inline-block; width:150px; font-size:12px; color:#888;">(' + dateStr + ')</span>' +
                                              '<button style="width:auto; padding: 6px 15px; font-size:13px; background:#28a745;" onclick="document.getElementById(\'versionId\').value=\'' + versionId + '\'; alert(\'ID: ' + versionId + ' (对应版本 ' + data.displayVersion + ') 已自动填入下载框！请直接点击第4步的开始下载！\')">🎯 选定此版本下载</button>';
            } else {
                btn.innerText = "❌ 解析失败，苹果未返回";
                btn.disabled = false;
            }
        }

        async function downloadApp() {
            logOut("【正在请求下载授权并建立下载链路，请耐心等待进度条完成】...\n----------------------------------------\n");
            let args = ['download', '--purchase', '-b', v('bundleId'), '--external-version-id', v('versionId')];
            if (v('savePath')) args.push('-o', v('savePath'));
            let text = await runBackend(args);
            appendLog(text);
        }
    </script>
</body>
</html>
`

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
				if err := json.NewDecoder(r.Body).Decode(&cmdArgs); err != nil || len(cmdArgs) == 0 {
					w.Write([]byte("❌ 参数解析失败"))
					return
				}

				hasFormat := false
				for _, a := range cmdArgs {
					if a == "--format" { hasFormat = true; break }
				}

				exePath, err := os.Executable()
				if err != nil { exePath = "ipatool" }

				execCmd := exec.Command(exePath, cmdArgs...)
				if !hasFormat {
					execCmd.Args = append(execCmd.Args, "--format", "text")
				}
				
				execCmd.Args = append(execCmd.Args, "--non-interactive", "--keychain-passphrase", "web-passphrase-123")

				out, _ := execCmd.CombinedOutput()
				w.Write(out)
			})

			dependencies.Logger.Log().Msg("🎉 可视化面板已启动！请打开浏览器访问: http://127.0.0.1:8080")
			return http.ListenAndServe(":8080", nil)
		},
	}
	
	return cmd
}
