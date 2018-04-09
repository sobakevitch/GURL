package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"

	flag "github.com/spf13/pflag"
	"golang.org/x/crypto/pkcs12"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/cheggaaa/pb.v1"
)

var (
	useragent  = "GURL"
	p12Path    string
	url        string
	output     string
	caroot     string
	errlog     *log.Logger
	showHeader bool
	status     bool
)

func askPassword(label string) string {
	fmt.Print(label)
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		panic(err)
	}
	password := string(bytePassword)
	fmt.Println()
	return password
}

func getPemData(p12 []byte, password string) tls.Certificate {
	blocks, err := pkcs12.ToPEM(p12, password)
	if err != nil {
		panic(err)
	}

	var pemData []byte
	for _, b := range blocks {
		pemData = append(pemData, pem.EncodeToMemory(b)...)
	}
	cert, err := tls.X509KeyPair(pemData, pemData)
	if err != nil {
		panic(err)
	}
	return cert
}

func printHeader(proto, status string, header http.Header) {
	fmt.Printf("%s %s\n", proto, status)
	for k, v := range header {
		fmt.Printf("%s: %s\n", k, strings.Join(v, ","))
	}
	fmt.Println()
}

func copyRemoteFile(header http.Header, body io.Reader) {
	var rd io.Reader
	var out io.Writer
	var err error

	if output != "" && status {
		size, err := strconv.ParseInt(header.Get("Content-Length"), 10, 64)
		if err != nil {
			panic(err)
		}
		bar := pb.New64(size)
		bar.Start()
		rd = bar.NewProxyReader(body)
	} else {
		rd = body
	}

	if output != "" {
		out, err = os.OpenFile(output, os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			panic(err)
		}
	} else {
		out = os.Stdout
	}
	io.Copy(out, rd)
}

func init() {
	errlog = log.New(os.Stderr, "", 0)

	flag.BoolVarP(&showHeader, "header", "i", false, "show HTTP headers")
	flag.BoolVarP(&status, "status", "s", false, "show a progress bar, only valid with --output")
	flag.StringVarP(&useragent, "user-agent", "A", "", "Set the User-Agent string (default: GURL)")
	flag.StringVar(&p12Path, "p12", "", "Path to a p12 file")
	flag.StringVar(&caroot, "caroot", "", "CA root")
	flag.StringVarP(&output, "output", "o", "", "Output file to write the content of the retrieved URL")

	flag.Parse()
	if flag.NArg() > 0 {
		url = flag.Args()[0]
	} else {
		errlog.Println("Usage: missing URL")
		os.Exit(1)
	}
}

func main() {

	var config tls.Config
	if p12Path != "" {
		p12, err := ioutil.ReadFile(p12Path)
		if err != nil {
			panic(err)
		}
		password := askPassword("PKCS#12 password: ")
		cert := getPemData(p12, password)
		config.Certificates = []tls.Certificate{cert}
	}
	if caroot != "" {
		caPool := x509.NewCertPool()
		severCert, err := ioutil.ReadFile(caroot)
		if err != nil {
			panic(err)
		}
		caPool.AppendCertsFromPEM(severCert)
		config.RootCAs = caPool
	}

	tr := &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: &config,
	}
	client := &http.Client{Transport: tr}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("User-Agent", useragent)

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if showHeader {
		printHeader(resp.Proto, resp.Status, resp.Header)
	}

	copyRemoteFile(resp.Header, resp.Body)

}
