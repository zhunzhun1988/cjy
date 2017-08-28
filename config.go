package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/tealeg/xlsx"
)

const (
	NameIndex        = "商品名称"
	PriceIndex       = "销售价"
	MessageIndex     = "上传状态"
	KuCunIndex       = "库存"
	ProductTypeIndex = "商品分类"
	ImagePathIndex   = "图片位置"
)

type UpLoadItem struct {
	Name       string
	Price      string
	Type       string
	KuCun      string
	MajorImage []string
	DitalImage []string
}

type Config struct {
	path           string
	file           *xlsx.File
	musthaveColume []string
	productDirs    []string
	indexMap       map[string]int
	typeMap        map[string]string
}

func ReadLine(fileName string, handler func(string)) error {
	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	buf := bufio.NewReader(f)
	for {
		line, err := buf.ReadString('\n')
		line = strings.TrimSpace(line)
		handler(line)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
	return nil
}

func (config *Config) lineTypeAdd(line string) {
	strs := strings.Split(line, ",")
	if len(strs) == 2 {
		config.typeMap[strs[1]] = strs[0]
	}
}

func checkConfig(config *Config) error {
	if isFileExist(path.Join(path.Dir(config.path), "type")) == false {
		return fmt.Errorf("type文件找不到")
	}
	if config.file == nil {
		return fmt.Errorf("无效文件")
	}
	if len(config.file.Sheets) == 0 {
		return fmt.Errorf("找不到Sheet")
	}
	if len(config.file.Sheets[0].Rows) < 2 {
		return fmt.Errorf("没有商品数据")
	}
	checkMap := make(map[string]bool)
	for index, titleCell := range config.file.Sheets[0].Rows[0].Cells {
		if config.isValidColume(titleCell.Value) {
			checkMap[titleCell.Value] = true
			config.indexMap[titleCell.Value] = index
		}
	}
	if len(checkMap) == len(config.musthaveColume) {
		return nil
	}
	for _, c := range config.musthaveColume {
		if exist, _ := checkMap[c]; exist == false {
			return fmt.Errorf("没有定义%s", c)
		}
	}
	return fmt.Errorf("未知错位")
}

func (config *Config) getTypeCode(typeName string) string {
	code, _ := config.typeMap[typeName]
	return code
}
func (config *Config) isValidColume(name string) bool {
	for _, n := range config.musthaveColume {
		if n == name {
			return true
		}
	}
	return false
}
func ReadConfig(filepath string) (*Config, error) {
	config := &Config{
		path:           filepath,
		indexMap:       make(map[string]int),
		typeMap:        make(map[string]string),
		productDirs:    getSubDirs(path.Join(path.Dir(filepath), "pic")),
		musthaveColume: []string{NameIndex, PriceIndex, MessageIndex, KuCunIndex, ProductTypeIndex, ImagePathIndex},
	}
	file, err := xlsx.OpenFile(filepath)
	if err != nil {
		return nil, err
	}
	config.file = file

	if errCheck := checkConfig(config); errCheck != nil {
		return nil, errCheck
	}

	ReadLine(path.Join(path.Dir(filepath), "type"), config.lineTypeAdd)
	//ssfmt.Printf("typemap=%v\n", config.typeMap)
	return config, nil
}

func (config *Config) IsRowValid(row int) (bool, error) {
	if len(config.file.Sheets[0].Rows) < row || row <= 0 {
		return false, fmt.Errorf("该行不存在")
	}

	// check name
	if nameIndex, exist := config.indexMap[NameIndex]; nameIndex < 0 || exist == false {
		return false, fmt.Errorf("名称不存在")
	} else if len(config.file.Sheets[0].Rows[row].Cells) <= nameIndex {
		return false, fmt.Errorf("名称不存在")
	} else if config.file.Sheets[0].Rows[row].Cells[nameIndex].Value == "" {
		return false, fmt.Errorf("名称为空")
	}

	// check price
	if priceIndex, exist := config.indexMap[PriceIndex]; priceIndex < 0 || exist == false {
		return false, fmt.Errorf("价格不存在")
	} else if len(config.file.Sheets[0].Rows[row].Cells) <= priceIndex {
		return false, fmt.Errorf("价格不存在")
	} else if config.file.Sheets[0].Rows[row].Cells[priceIndex].Value == "" {
		return false, fmt.Errorf("价格为空")
	} else if v, err := strconv.ParseFloat(config.file.Sheets[0].Rows[row].Cells[priceIndex].Value, 64); v <= 0.0 && err != nil {
		return false, fmt.Errorf("价格不合法")
	}

	// check 库存
	if kuCunIndex, exist := config.indexMap[KuCunIndex]; kuCunIndex < 0 || exist == false {
		return false, fmt.Errorf("库存不存在")
	} else if len(config.file.Sheets[0].Rows[row].Cells) <= kuCunIndex {
		return false, fmt.Errorf("库存不存在")
	} else if config.file.Sheets[0].Rows[row].Cells[kuCunIndex].Value == "" {
		return false, fmt.Errorf("库存为空")
	} else if v, err := strconv.Atoi(config.file.Sheets[0].Rows[row].Cells[kuCunIndex].Value); err != nil || v < 0 {
		return false, fmt.Errorf("库存无效")
	}

	// check image path
	dirPath := path.Dir(config.path)
	if imagePathIndex, exist := config.indexMap[ImagePathIndex]; imagePathIndex < 0 || exist == false ||
		len(config.file.Sheets[0].Rows[row].Cells) <= imagePathIndex ||
		config.file.Sheets[0].Rows[row].Cells[imagePathIndex].Value == "" {
		nameIndex, _ := config.indexMap[NameIndex]
		defaultPath := getDefaultDir(config.productDirs, config.file.Sheets[0].Rows[row].Cells[nameIndex].Value)
		if isFileExist(defaultPath) == false {
			return false, fmt.Errorf("图片位置为空")
		} else {
			tmpPath := strings.Replace(defaultPath, path.Clean(path.Dir(config.path)), "", -1)
			if string(tmpPath[0:1]) == "/" || string(tmpPath[0:1]) == "\\" {
				tmpPath = tmpPath[1:]
			}

			config.SetImagePath(row, tmpPath)
		}
	} else if p := path.Join(dirPath, config.file.Sheets[0].Rows[row].Cells[imagePathIndex].Value); isFileExist(p) == false {
		return false, fmt.Errorf("图片位置无效")
	}

	// check type
	if productTypeIndex, exist := config.indexMap[ProductTypeIndex]; productTypeIndex < 0 || exist == false {
		return false, fmt.Errorf("商品类型不存在")
	} else if len(config.file.Sheets[0].Rows[row].Cells) <= productTypeIndex {
		return false, fmt.Errorf("商品类型不存在")
	} else if config.file.Sheets[0].Rows[row].Cells[productTypeIndex].Value == "" {
		return false, fmt.Errorf("商品类型为空")
	} else if config.getTypeCode(config.file.Sheets[0].Rows[row].Cells[productTypeIndex].Value) == "" {
		return false, fmt.Errorf("未知商品类型[%s]", config.file.Sheets[0].Rows[row].Cells[productTypeIndex].Value)
	}

	return true, nil
}

func (config *Config) GetName(row int) (string, error) {
	if ok, _ := config.IsRowValid(row); ok == false {
		return "", fmt.Errorf("row %d is invalid", row)
	}
	return config.getValue(row, NameIndex)
}

func (config *Config) GetPrice(row int) (string, error) {
	if ok, _ := config.IsRowValid(row); ok == false {
		return "", fmt.Errorf("row %d is invalid", row)
	}
	str, err := config.getValue(row, PriceIndex)
	if err != nil {
		return "", err
	}
	v, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%.1f", v), nil
}

func (config *Config) GetMsg(row int) (string, error) {
	if ok, _ := config.IsRowValid(row); ok == false {
		return "", fmt.Errorf("row %d is invalid", row)
	}
	return config.getValue(row, MessageIndex)
}
func (config *Config) GetType(row int) (string, error) {
	if ok, _ := config.IsRowValid(row); ok == false {
		return "", fmt.Errorf("row %d is invalid", row)
	}
	return config.getValue(row, ProductTypeIndex)
}

func (config *Config) GetKuCun(row int) (string, error) {
	if ok, _ := config.IsRowValid(row); ok == false {
		return "", fmt.Errorf("row %d is invalid", row)
	}
	return config.getValue(row, KuCunIndex)
}
func (config *Config) GetImagePath(row int) (string, error) {
	if ok, _ := config.IsRowValid(row); ok == false {
		return "", fmt.Errorf("row %d is invalid", row)
	}
	p, err := config.getValue(row, ImagePathIndex)
	if err != nil {
		return "", err
	}
	if p == "" {
		name, _ := config.GetName(row)
		p = getDefaultDir(config.productDirs, name)
	}
	return path.Join(path.Dir(config.path), p), nil
}

func (config *Config) GetRowNum() int {
	return len(config.file.Sheets[0].Rows) - 1
}

func (config *Config) getValue(row int, name string) (string, error) {
	index, exist := config.indexMap[name]
	if exist == false {
		return "", fmt.Errorf("not exist")
	}
	if len(config.file.Sheets[0].Rows[row].Cells) <= index {
		return "", nil
	}
	return config.file.Sheets[0].Rows[row].Cells[index].Value, nil
}

func (config *Config) SetImagePath(row int, path string) error {
	if len(config.file.Sheets[0].Rows) <= row {
		return fmt.Errorf("row not exist")
	}
	columeNum := len(config.file.Sheets[0].Rows[row].Cells)
	index, _ := config.indexMap[ImagePathIndex]
	for i := columeNum; i <= index; i++ {
		config.file.Sheets[0].Rows[row].AddCell()
	}
	config.file.Sheets[0].Rows[row].Cells[index].Value = path
	err := config.file.Save(config.path)
	return err
}

func (config *Config) SetMsg(row int, msg string) error {
	if len(config.file.Sheets[0].Rows) <= row {
		return fmt.Errorf("row not exist")
	}
	columeNum := len(config.file.Sheets[0].Rows[row].Cells)
	index, _ := config.indexMap[MessageIndex]
	for i := columeNum; i <= index; i++ {
		config.file.Sheets[0].Rows[row].AddCell()
	}
	config.file.Sheets[0].Rows[row].Cells[index].Value = msg
	err := config.file.Save(config.path)
	return err
}

func (config *Config) GetUploadItem(row int) (*UpLoadItem, error) {
	if ok, _ := config.IsRowValid(row); ok == false {
		return nil, fmt.Errorf("row %d is invalid", row)
	}
	name, _ := config.GetName(row)
	price, _ := config.GetPrice(row)
	productType, _ := config.GetType(row)
	kucun, _ := config.GetKuCun(row)
	imagePath, _ := config.GetImagePath(row)
	ret := &UpLoadItem{
		Name:       name,
		Price:      price,
		Type:       config.getTypeCode(productType),
		KuCun:      kucun,
		MajorImage: getMajorImagePaths(imagePath),
		DitalImage: getDitalImagePaths(imagePath),
	}
	return ret, nil
}

func getMajorImagePaths(dir string) []string {
	files := getDirFiles(dir)
	filter := make([]string, 0, len(files))
	for _, f := range files {
		name := path.Base(f)
		if strings.Contains(name, "封面") == true {
			filter = append(filter, f)
		}
	}
	sort.Sort(PathStr(filter))
	return filter
}
func getDitalImagePaths(dir string) []string {
	files := getDirFiles(dir)
	filter := make([]string, 0, len(files))
	for _, f := range files {
		name := path.Base(f)
		if strings.Contains(name, "封面") == false {
			filter = append(filter, f)
		}
	}
	sort.Sort(PathStr(filter))
	return filter
}
func getDirFiles(dir string) []string {
	entries, err := ioutil.ReadDir(dir)
	ret := make([]string, 0, len(entries))
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		if entry.IsDir() == false {
			ret = append(ret, path.Join(dir, entry.Name()))
		}
	}
	return ret
}
func getSubDirs(dir string) []string {
	entries, err := ioutil.ReadDir(dir)
	ret := make([]string, 0, len(entries))
	if err != nil {
		return []string{}
	}
	for _, entry := range entries {
		if entry.IsDir() == true {
			ret = append(ret, path.Join(dir, entry.Name()))
		}
	}
	return ret
}
func getDefaultDir(dirs []string, name string) string {
	for _, d := range dirs {
		if strings.Contains(name, path.Base(d)) {
			return d
		}
	}
	return ""
}

func isFileExist(p string) bool {
	var exist = true
	if _, err := os.Stat(p); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

type PathStr []string

func (ps PathStr) Len() int { return len(ps) }
func (ps PathStr) Less(i, j int) bool {
	stri := path.Base(ps[i])
	strj := path.Base(ps[j])
	return stri < strj
}
func (ps PathStr) Swap(i, j int) {
	ps[i], ps[j] = ps[j], ps[i]
}
