package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	usename     *string = flag.String("username", "", "login usename")
	password    *string = flag.String("password", "", "login password")
	configfile  *string = flag.String("configpath", "/home/adam/go/src/cjy/excel.xlsx", "excel path")
	isDebug     *bool   = flag.Bool("debug", true, "for debug , only create the excel row one")
	isCheckOnly *bool   = flag.Bool("check", true, "for check the excel only")
)

func main() {
	flag.Parse()
	// step 1 check parameter
	if usename == nil || *usename == "" || password == nil || *password == "" {
		fmt.Printf("请指定用户名和密码\n")
		os.Exit(-1)
	}
	if configfile == nil || *configfile == "" {
		fmt.Printf("请输入配置excel文件\n")
		os.Exit(-1)
	} else if isFileExist(*configfile) == false {
		fmt.Printf("%s,配置excel文件找不到\n", *configfile)
		os.Exit(-1)
	}
	fmt.Printf("用户名： \t%s\n密    码： \t%s\nExcel文件： %s\n调试:\t%t\n检测：\t%t\n",
		*usename, *password, *configfile, *isDebug, *isCheckOnly)

	// step2 start login server
	fmt.Printf("开始登录服务器...\n")
	client, errLogin := Login(*usename, *password)
	if errLogin != nil {
		fmt.Printf("登录出错:%v\n", errLogin)
		os.Exit(-1)
	}
	fmt.Printf("服务器登录成功\n")

	// step3 parse excel file
	fmt.Printf("开始解析execl文件:%s\n", *configfile)
	config, err := ReadConfig(*configfile)
	if err != nil {
		fmt.Printf("读取excel错误：%v\n", err)
		os.Exit(-1)
	}
	// step4 do job
	rowNum := config.GetRowNum()
	if isDebug != nil && *isDebug == true {
		rowNum = 1
	}
	fmt.Printf("总共有%d行[%d]\n", rowNum, config.GetRowNum())
	var okNum, errNum, skipNum int
	for i := 1; i <= rowNum; i++ {
		fmt.Printf("开始处理行[%03d]: ", i)
		if ok, err := config.IsRowValid(i); ok == false {
			fmt.Printf("处理出错（IsRowValid）:%v", err)
			config.SetMsg(i, err.Error())
			errNum++
		} else if msg, err := config.GetMsg(i); err == nil && msg == "OK" {
			fmt.Printf("已经处理过")
			skipNum++
		} else {
			item, err := config.GetUploadItem(i)
			if err != nil || item == nil {
				fmt.Printf("处理出错（GetUploadItem）:%v", err)
				config.SetMsg(i, err.Error())
				errNum++
			} else {
				errCreate := client.CreateProduct(item, *isCheckOnly)
				if errCreate != nil {
					config.SetMsg(i, errCreate.Error())
					fmt.Printf("处理出错（CreateProduct）:%v", errCreate)
					errNum++
				} else {
					if *isCheckOnly {
						fmt.Printf("Check OK")
						config.SetMsg(i, "Check OK")
					} else {
						fmt.Printf("OK")
						config.SetMsg(i, "OK")
					}
					okNum++
				}
			}
		}
		fmt.Printf("\n")
	}
	fmt.Printf("全部处理完成： 出错行数[%d], 跳过行数[%d], 成功行数[%d], 总共[%d]\n", errNum, skipNum, okNum, rowNum)
}
