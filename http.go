package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
)

const (
	server_addr      = "http://youwuku.cn/egou/index.php/shangjia/login/loginsubnew?version=youwuku"
	upload_addr      = "http://youwuku.cn/egou/uploadify/UploadImg.php"
	deleteImage_addr = "http://youwuku.cn/egou/index.php/shangjia/commoncomponent/delpic?"
	create_addr      = "http://youwuku.cn/egou/index.php/shangjia/prd/saveprdinfo"
)

type HttpRet struct {
	Code int
	Msg  string
}
type Client struct {
	client *http.Client
}
type Jar struct {
	lk      sync.Mutex
	cookies map[string][]*http.Cookie
}

func newJar() *Jar {
	jar := new(Jar)
	jar.cookies = make(map[string][]*http.Cookie)
	return jar
}

// SetCookies handles the receipt of the cookies in a reply for the
// given URL.  It may or may not choose to save the cookies, depending
// on the jar's policy and implementation.
func (jar *Jar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	jar.lk.Lock()
	jar.cookies[u.Host] = cookies
	jar.lk.Unlock()
}

// Cookies returns the cookies to send in a request for the given URL.
// It is up to the implementation to honor the standard cookie use
// restrictions such as in RFC 6265.
func (jar *Jar) Cookies(u *url.URL) []*http.Cookie {
	return jar.cookies[u.Host]
}

func Login(username, password string) (*Client, error) {
	client := &Client{
		client: &http.Client{
			Jar: newJar(),
		},
	}
	url := fmt.Sprintf("%s&inputEmail=%s&inputpassword=%s&rememberpd=keepinfo&version=youwuku&back_url=http%%3A%%2F%%2Fyouwuku.cn%%2Fegou%%2Findex.php%%2Fshangjia%%2Faccount%%3Fver%%3D2&account=&nonce=",
		server_addr, username, password)
	res, err := client.client.PostForm(url, nil)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", res.Status)
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	ret := &HttpRet{}
	err = json.Unmarshal(body, ret)
	if err != nil {
		return nil, err
	}
	if ret.Msg != "" {
		return nil, fmt.Errorf("%s", ret.Msg)
	}
	return client, nil
}

func getImageTypeStr(imagepath string) string {
	if strings.HasSuffix(strings.ToLower(imagepath), ".png") {
		return "image/png"
	} else if strings.HasSuffix(strings.ToLower(imagepath), ".jpg") {
		return "image/jpg"
	}
	return ""
}

func getFileSizeStr(imagepath string) string {
	info, err := os.Stat(imagepath)
	if err != nil {
		return ""
	} else {
		return fmt.Sprintf("%d", info.Size())
	}
}
func (c *Client) DeleteImage(id, image string) error {
	url := fmt.Sprintf("%sid=%s&pic_name=%s-", deleteImage_addr, id, image)
	res, err := c.client.Get(url)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", res.Status)
	}
	buf := make([]byte, 0, 1024)
	res.Body.Read(buf)
	fmt.Printf("delete:%s\n", string(buf))
	return nil
}

func (c *Client) UploadImage(imagepath string) (string, error) {
	typeStr := getImageTypeStr(imagepath)
	if typeStr == "" {
		return "", fmt.Errorf("unknow image type")
	}
	sizeStr := getFileSizeStr(imagepath)
	if sizeStr == "" {
		return "", fmt.Errorf("get file size error")
	}
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	fw, err := w.CreateFormField("is_weialbum")
	if err != nil {
		return "", err
	}
	if _, err = fw.Write([]byte("1")); err != nil {
		return "", err
	}

	if fw, err = w.CreateFormField("album"); err != nil {
		return "", err
	}
	if _, err = fw.Write([]byte("35707")); err != nil {
		return "", err
	}

	if fw, err = w.CreateFormField("id"); err != nil {
		return "", err
	}
	if _, err = fw.Write([]byte("WU_FILE_1")); err != nil {
		return "", err
	}

	if fw, err = w.CreateFormField("name"); err != nil {
		return "", err
	}
	if _, err = fw.Write([]byte(path.Base(imagepath))); err != nil {
		return "", err
	}

	if fw, err = w.CreateFormField("type"); err != nil {
		return "", err
	}
	if _, err = fw.Write([]byte(typeStr)); err != nil {
		return "", err
	}

	if fw, err = w.CreateFormField("size"); err != nil {
		return "", err
	}
	if _, err = fw.Write([]byte(sizeStr)); err != nil {
		return "", err
	}

	// Add your image file
	f, err := os.Open(imagepath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if fw, err = CreateFormFile(w, "Filedata", path.Base(imagepath), typeStr); err != nil {
		return "", err
	}
	if _, err = io.Copy(fw, f); err != nil {
		return "", err
	}

	w.Close()

	// Now that you have a form, you can submit it to your handler.
	req, err := http.NewRequest("POST", upload_addr, &b)
	req.Header["Origin"] = []string{"http://youwuku.cn"}
	if err != nil {
		return "", err
	}
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", w.FormDataContentType())

	res, err := c.client.Do(req)
	if err != nil {
		return "", err
	}

	// Check the response
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", res.Status)
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	ret := &HttpRet{}
	err = json.Unmarshal(body, ret)
	if err != nil {
		return "", err
	}
	return ret.Msg, nil
}
func getFormData(item *UpLoadItem) url.Values {
	ret := make(url.Values)
	ret["prddata[prdtitle]"] = []string{item.Name}
	ret["prddata[prdspec]"] = []string{"1"}
	ret["prddata[prdtype]"] = []string{"1"}
	ret["prddata[proname]"] = []string{""}
	ret["prddata[prdprice]"] = []string{item.Price}
	ret["prddata[prdnum]"] = []string{item.KuCun}
	ret["prddata[prdcode]"] = []string{""}
	ret["prddata[catsname]"] = []string{item.Type}
	ret["prddata[prdimgs][]"] = item.MajorImage
	ret["prddata[prddesc]"] = []string{UrlEncode(getDetailImageXml(item.DitalImage))}
	ret["prddata[LogisProvince]"] = []string{"上海"}
	ret["prddata[LogisCity]"] = []string{"上海市"}
	ret["prddata[prdtransport]"] = []string{"2"}
	ret["prddata[prdexpress]"] = []string{"0"}
	ret["prddata[prdexpress1]"] = []string{""}
	ret["prddata[prdsale]"] = []string{"unsale"}
	return ret
}
func getUrlStr(item *UpLoadItem) (string, error) {
	var url string
	url = "prddata[prdtitle]=" + item.Name
	url += "&prddata[prdspec]=1"
	url += "&prddata[prdtype]=1"
	url += "&prddata[proname]="
	url += "&prddata[prdprice]=" + item.Price
	url += "&prddata[prdnum]=" + item.KuCun
	url += "&prddata[prdcode]="
	url += "&prddata[catsname]=,e101314,e101307"

	url += "&prddata[prdimgs][]=" + item.MajorImage[0]
	url += "&prddata[prddesc]=" + UrlEncode(getDetailImageXml(item.DitalImage))
	url += "&prddata[LogisProvince]=上海"
	url += "&prddata[LogisCity]=上海市"
	url += "&prddata[prdtransport]=2"
	url += "&prddata[prdexpress]=0"
	url += "&prddata[prdexpress1]="
	url += "&prddata[prdsale]=unsale"
	return url, nil
}
func getDetailImageXml(images []string) string {
	var ret string
	for _, image := range images {
		ret += fmt.Sprintf("<img src=\"%s\"/>", image)
	}
	return string("<p>") + ret + string("</p>")
}
func (c *Client) CreateProduct(item *UpLoadItem) error {
	if len(item.MajorImage) == 0 {
		return fmt.Errorf("没有定义封面图")
	} else if len(item.DitalImage) == 0 {
		return fmt.Errorf("没有定义详情图")
	}

	majorImage := make([]string, 0, len(item.MajorImage))
	for _, image := range item.MajorImage {
		serverUrl, err := c.tryUploadImage(5, image)
		if err != nil {
			return fmt.Errorf("updatel image[%s]:%v", image, err)
		}
		majorImage = append(majorImage, serverUrl)
	}

	detailImage := make([]string, 0, len(item.DitalImage))
	for _, image := range item.DitalImage {
		serverUrl, err := c.tryUploadImage(5, image)
		if err != nil {
			return fmt.Errorf("updatel image[%s]:%v", image, err)
		}
		detailImage = append(detailImage, serverUrl+getMajorImageSuffix(serverUrl))
	}
	item.DitalImage = detailImage
	item.MajorImage = majorImage

	res, errPost := c.client.PostForm(create_addr, getFormData(item))
	if errPost != nil {
		return errPost
	}
	// Check the response
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", res.Status)
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	ret := &HttpRet{}
	err = json.Unmarshal(body, ret)
	if err != nil {
		return err
	}
	return nil

	/*	if str, err := getUrlStr(item); err != nil {
			return err
		} else {

			url := create_addr + string("?") + UrlEncode(str)

			fmt.Printf("\n\n %s \n\n", url)
			res, errPost := c.client.PostForm(url, nil)
			if errPost != nil {
				return errPost
			}
			// Check the response
			if res.StatusCode != http.StatusOK {
				return fmt.Errorf("bad status: %s", res.Status)
			}
			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				return err
			}
			ret := &HttpRet{}
			err = json.Unmarshal(body, ret)
			if err != nil {
				return err
			}
			fmt.Printf("code=%d,body=%s\n", ret.Code, ret.Msg)
			return nil
		}
		return nil*/
}

func getMajorImageSuffix(image string) string {
	return "_800x800.jpg"
}
func (c *Client) tryUploadImage(count int, image string) (string, error) {
	var lasterr error
	for i := 0; i < count; i++ {
		serverUrl, err := c.UploadImage(image)
		if err == nil && serverUrl != "" {
			return serverUrl, err
		}
		lasterr = err
	}
	return "", lasterr
}
func CreateFormFile(w *multipart.Writer, fieldname, filename string, typestr string) (io.Writer, error) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
			escapeQuotes(fieldname), escapeQuotes(filename)))
	h.Set("Content-Type", typestr)
	return w.CreatePart(h)
}

func (c *Client) Test() {
	res, errr := c.client.Get("http://127.0.0.1:8080/")
	body, _ := ioutil.ReadAll(res.Body)
	fmt.Printf("%v,%v\n", string(body), errr)

}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

func UrlEncode(str string) string {
	bytes := []byte(str)
	var ret string
	for _, b := range bytes {
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') {
			ret += string(b)
		} else {
			ret += fmt.Sprintf("%%%X", b)
		}
	}
	return ret
}
