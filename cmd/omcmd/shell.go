/*
Copyright © 2020 0x1un <aumujun@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"io"
	"omtools/adtools"
	"omtools/zbxtools"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/chzyer/readline"
	"github.com/spf13/cobra"
)

const (
	zbxUrl      = "http://%s/api_jsonrpc.php"
	cmdNotFound = "omtools: command not found: %s\n"
)

var (
	mode        = ""
	sessionInfo = map[string]string{}
)

// shellCmd represents the shell command
var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "交互模式",
	Long:  `进入交互模式`,
	Run:   shellcmd,
}

var destory = func() {
	if ad != nil {
		ad.BuiltinConn().Close()
		ad = nil
	}
}

func init() {
	rootCmd.AddCommand(shellCmd)
}

func shellcmd(cmd *cobra.Command, args []string) {

	// set line prompt
	l, err := readline.NewEx(&readline.Config{
		Prompt:              "\033[31momtools »\033[0m ",
		HistoryFile:         "./readline.tmp",
		AutoComplete:        completer,
		InterruptPrompt:     "^C",
		EOFPrompt:           "exit",
		HistorySearchFold:   true,
		FuncFilterInputRune: filterInput,
	})
	if err != nil {
		panic(err)
	}
	defer l.Close()

	setPasswordCfg := l.GenPasswordConfig()
	setPasswordCfg.SetListener(func(line []rune, pos int, key rune) (newLine []rune, newPos int, ok bool) {
		l.SetPrompt(fmt.Sprintf("Enter password(%v): ", len(line)))
		l.Refresh()
		return nil, 0, false
	})

	log.SetOutput(l.Stderr())
	for {
		line, err := l.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}
		if line == "bye" || line == "exit" {
			goto exit
		}

		line = strings.TrimSpace(line)
		switch line {
		case "re con zbx":
			if mode == "zbx" && zbx != nil && len(sessionInfo) != 0 {
				zbx = zbxtools.NewZbxTool(fmt.Sprintf(zbxUrl, sessionInfo["zbxAddr"]), sessionInfo["zbxUser"], sessionInfo["zbxPwd"])
			}

		case "re con ad":
			if mode == "ad" && ad != nil && len(sessionInfo) != 0 {
				ad, err = adtools.NewADTools(sessionInfo["adAddr"], sessionInfo["adUser"], sessionInfo["adPwd"])
				if err != nil {
					fmt.Printf("failed connect to %s, err:%s\n", sessionInfo["adAddr"], err.Error())
				}
			}
		case "go zbx":
			url, username, password := getInputWithPromptui("")
			zbx = zbxtools.NewZbxTool(fmt.Sprintf(zbxUrl, url), username, password)
			mode = "zbx"
			sessionInfo["zbxAddr"] = url
			sessionInfo["zbxUser"] = username
			sessionInfo["zbxPwd"] = password
		case "go ad":
			url, buser, bpass := getInputWithPromptui("ad")
			ad, err = adtools.NewADTools(url, buser, bpass)
			if err != nil {
				fmt.Printf("failed connect to %s, err:%s\n", url, err.Error())
			}
			mode = "ad"
			sessionInfo["adAddr"] = url
			sessionInfo["adUser"] = buser
			sessionInfo["adPwd"] = bpass
		}
		if mode == "zbx" && zbx != nil {
			zbxCmdHandler(line)
		}
		if mode == "ad" && ad != nil {
			adCmdHandler(line)
		}
	}
exit:
	destory()
}

func adCmdHandler(line string) {
	switch {
	case line == "add single user":
		disname, username, org, pwd, des, disabled := getUserInfo()
		err := ad.AddUser(disname, username, org, pwd, des, disabled)
		if err != nil {
			fmt.Println(err)
			return
		}
	case strings.HasPrefix(line, "add user from "):
		// TODO: 检查文件路径合法性
		l := line[14:]
		if len(l) == 0 {
			println("请输入文件路径")
			return
		}
		for _, e := range ad.AddUserMultiple(l, getOuPath(), false).Errors {
			fmt.Println(e)
		}
	case strings.HasPrefix(line, "del user with "):
		l := line[14:]
		if len(l) == 0 {

		}
	case strings.HasPrefix(line, "query info "):
		l := ""
		if l = line[11:]; l == "all" {
			l = "*"
		}
		res, err := ad.GetUserInfoTable(l)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println(res)

	case strings.HasPrefix(line, "dis ") || strings.HasPrefix(line, "ena "):
		l := line[4:]
		if len(l) != 0 {
			l = strings.TrimSpace(l)
			err := changeStatus(l, line[:3] == "dis")
			if err != nil {
				fmt.Println(err.Error())
			}
		}
	case strings.HasPrefix(line, "del user "):
		if c := line[9:]; len(c) != 0 {
			// TODO: search user and list them

			// remove user
		}
	case line == "go ad":
		fmt.Println("connect to ad server...")
	case line == "re con ad":
		fmt.Println("reconnect to ad server...")
	case len(line) == 0:
	default:
		fmt.Printf(cmdNotFound, line)
	}
}

func zbxCmdHandler(line string) {
	switch {
	case strings.HasPrefix(line, "list "):
		switch line[5:] {
		case "host":
			cmdMap[line[:4]]("", line[5:])
		case "group":
			cmdMap[line[:4]]("", line[5:])
		}
	// query [host] by [key]
	case strings.HasPrefix(line, "query "):
		subcmd := line[6:]
		subList := strings.Split(subcmd, " ")
		if len(subList) >= 3 {
			if subList[1] == "by" {
				cmdMap[line[:5]](subList[2], subList[0])
			}
		}
	case strings.HasPrefix(line, "cfg "):
		subcmd := line[4:]
		subList := strings.Split(subcmd, " ")
		if len(subList) >= 2 {
			if subList[0] == "export" {
				cmdMap[line[:3]](subList[1], "")
			}
		}
	case line == "go zbx":
		println("connect to zabbix server...")
	case line == "re con zbx":
		println("reconnect to zabbix server...")
	case len(line) == 0:
	default:
		fmt.Printf(cmdNotFound, line)
	}
}
