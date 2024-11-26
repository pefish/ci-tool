package ci_manager

import (
	"fmt"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/pefish/ci-tool/pkg/global"
	"github.com/pefish/ci-tool/pkg/util"
	go_file "github.com/pefish/go-file"
	i_logger "github.com/pefish/go-interface/i-logger"
	go_shell "github.com/pefish/go-shell"
	go_time "github.com/pefish/go-time"
	"github.com/pkg/errors"
)

var CiManager *CiManagerType

type CiManagerType struct {
	logs   sync.Map // map[string]string
	logger i_logger.ILogger
}

func NewCiManager(logger i_logger.ILogger) *CiManagerType {
	c := &CiManagerType{
		logger: logger,
	}
	return c
}

func (c *CiManagerType) Logs(fullName string) string {
	d, ok := c.logs.Load(fullName)
	if !ok {
		return ""
	} else {
		return d.(string)
	}
}

func (c *CiManagerType) StartCi(
	env,
	repo,
	fetchCodeKey,
	gitUsername,
	srcPath,
	config,
	fullName string,
	imageName string,
	port uint64,
	dockerNetwork string,
) {
	c.logs.Delete(fullName)
	logger := c.logger.CloneWithPrefix(fullName)
	logger.InfoF("<%s> running...\n", fullName)
	err := c.startCi(
		logger,
		env,
		repo,
		fetchCodeKey,
		srcPath,
		config,
		fullName,
		imageName,
		port,
		dockerNetwork,
	)
	if err != nil {
		c.logs.Store(fullName, err.Error())
		util.Alert(
			c.logger,
			fmt.Sprintf("[ERROR] <%s> <%s> 环境发布失败。\n%+v", fullName, env, err),
		)
		logger.ErrorF("<%s> failed!!! %+v", fullName, err)
		return
	}

	util.Alert(
		c.logger,
		fmt.Sprintf("[INFO] <%s> <%s> 环境发布成功。", fullName, env),
	)

	logger.InfoF("<%s> done!!!", fullName)
}

func (c *CiManagerType) startCi(
	logger i_logger.ILogger,
	env,
	repo,
	fetchCodeKey,
	srcPath,
	config,
	fullName string,
	imageName string,
	port uint64,
	dockerNetwork string,
) error {

	if env != "test" && env != "prod" {
		return errors.New("Env is illegal.")
	}

	branch := "test"
	if env == "prod" {
		branch = "main"
	}

	if strings.HasPrefix(srcPath, "~") {
		srcPath = "${HOME}" + srcPath[1:]
	}

	if _, ok := global.GlobalData.StartLogTime[fullName]; !ok {
		global.GlobalData.StartLogTime[fullName] = time.Now()
	}

	logsPath := path.Join(global.Command.DataDir, "logs", fullName)
	err := go_file.AssertPathExist(logsPath)
	if err != nil {
		return err
	}

	script := fmt.Sprintf(
		`
#!/bin/bash
set -euxo pipefail

src="`+srcPath+`"

# 检查源代码目录是否存在
if [ ! -d "$src" ]; then
    echo "源代码目录 '$src' 不存在，正在克隆仓库..."
    git clone%s "`+repo+`" "$src"
    
    if [ $? -eq 0 ]; then
        echo "克隆成功！"
    else
        echo "克隆失败，请检查 Git 仓库 URL 和网络连接。"
        exit 1
    fi
fi

cd ${src}
git config core.sshCommand "ssh -i `+fetchCodeKey+`"
git reset --hard && git clean -d -f . && git pull && git checkout `+branch+` && git pull

imageName="`+imageName+`:$(git rev-parse --short HEAD)"

if [[ "$(sudo docker images -q ${imageName} 2> /dev/null)" == "" ]]; then
  sudo docker build --build-arg APP_ENV=`+env+` -t ${imageName} .
fi

containerName="`+fullName+`-`+env+`"

sudo docker stop ${containerName}

containerId=$(docker inspect ${containerName} | grep '"Id"' | head -1 | awk -F '"' '{print $4}')

logPath="/var/lib/docker/containers/$containerId/${containerId}-json.log"

backupLogDir="`+logsPath+`"

sudo cat ${logPath} >> ${backupLogDir}/current.log

echo "日志已备份到 $backupLogDir"

%s

sudo docker rm ${containerName}

# 创建一个临时文件
TEMP_FILE=$(mktemp)

echo "`+config+`" > "$TEMP_FILE"

sudo docker run --name ${containerName} --env-file "$TEMP_FILE" -d %s%s ${imageName}

# 删除临时文件
rm "$TEMP_FILE"

`,
		func() string {
			if fetchCodeKey == "" {
				return ""
			} else {
				return fmt.Sprintf(` --config core.sshCommand="ssh -i %s"`, fetchCodeKey)
			}
		}(),
		func() string {
			if time.Since(global.GlobalData.StartLogTime[fullName]) > 10*24*time.Hour {
				now := time.Now()
				global.GlobalData.StartLogTime[fullName] = now
				return fmt.Sprintf(
					`
mv ${backupLogDir}/current.log ${backupLogDir}/%s_%s.log

echo "日志已打包"
`,
					go_time.TimeToStr(global.GlobalData.StartLogTime[fullName], "0000-00-00 00:00:00"),
					go_time.TimeToStr(now, "0000-00-00 00:00:00"),
				)
			} else {
				return ""
			}
		}(),
		func() string {
			if port == 0 {
				return ""
			} else {
				return fmt.Sprintf(" -p %d:8000", port)
			}
		}(),
		func() string {
			if dockerNetwork == "" {
				return ""
			} else {
				return fmt.Sprintf(` --network %s`, dockerNetwork)
			}
		}(),
	)
	cmd := exec.Command("/bin/bash", "-c", script)

	resultChan := make(chan string)
	go func() {
		for {
			select {
			case r := <-resultChan:
				logger.Info(r)
				d, ok := c.logs.Load(fullName)
				if !ok {
					c.logs.Store(fullName, r)
				} else {
					c.logs.Store(fullName, d.(string)+r+"\n")
				}
			}
		}
	}()
	err = go_shell.ExecForResultLineByLine(cmd, resultChan)
	if err != nil {
		return err
	}

	return nil
}
