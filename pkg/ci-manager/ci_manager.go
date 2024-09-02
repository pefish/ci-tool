package ci_manager

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/pefish/ci-tool/pkg/util"
	i_logger "github.com/pefish/go-interface/i-logger"
	go_logger "github.com/pefish/go-logger"
	go_shell "github.com/pefish/go-shell"
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
	port uint64,
	lokiUrl string,
	dockerNetwork string,
	alertTgToken string,
	alertGroupId string,
) {
	c.logs.Delete(fullName)
	logger := c.logger.CloneWithPrefix(fullName)
	logger.InfoF("<%s> running...\n", fullName)
	err := c.startCi(
		logger,
		env,
		repo,
		fetchCodeKey,
		gitUsername,
		srcPath,
		config,
		fullName,
		port,
		lokiUrl,
		dockerNetwork,
	)
	if err != nil {
		c.logs.Store(fullName, err.Error())
		util.Alert(
			go_logger.Logger,
			alertTgToken,
			alertGroupId,
			fmt.Sprintf("[ERROR] <%s> <%s> 环境发布失败。\n%+v", fullName, env, err),
		)
		logger.ErrorF("<%s> failed!!! %+v", fullName, err)
		return
	}

	util.Alert(
		go_logger.Logger,
		alertTgToken,
		alertGroupId,
		fmt.Sprintf("[INFO] <%s> <%s> 环境发布成功。", fullName, env),
	)

	logger.InfoF("<%s> done!!!", fullName)
}

func (c *CiManagerType) startCi(
	logger i_logger.ILogger,
	env,
	repo,
	fetchCodeKey,
	gitUsername,
	srcPath,
	config,
	fullName string,
	port uint64,
	lokiUrl string,
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

dockerBaseName="`+fullName+`"

imageName="${dockerBaseName}:$(git rev-parse --short HEAD)"

if [[ "$(sudo docker images -q ${imageName} 2> /dev/null)" == "" ]]; then
  sudo docker build --build-arg APP_ENV=`+env+` -t ${imageName} .
fi

containerName="${dockerBaseName}-`+env+`"

sudo docker stop ${containerName} && sudo docker rm ${containerName}

# 创建一个临时文件
TEMP_FILE=$(mktemp)

echo "`+config+`" > "$TEMP_FILE"

sudo docker run --name ${containerName} --env-file "$TEMP_FILE" -d %s%s%s ${imageName}

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
			if port == 0 {
				return ""
			} else {
				return fmt.Sprintf(" -p %d:8000", port)
			}
		}(),
		func() string {
			if lokiUrl == "" {
				return ""
			} else {
				return fmt.Sprintf(` --log-driver=loki --log-opt loki-url="%s" --log-opt loki-retries=5 --log-opt loki-batch-size=400`, lokiUrl)
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
	err := go_shell.ExecForResultLineByLine(cmd, resultChan)
	if err != nil {
		return err
	}

	return nil
}
