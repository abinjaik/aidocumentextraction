package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/textract"
	"github.com/spf13/viper"
)

var textractSession *textract.Textract

func init() {

	awsaccesskey := viperEnvVariable("awsaccesskey")
	awssecret := viperEnvVariable("awssecret")
	awstoken := viperEnvVariable("awstoken")
	awsregion := viperEnvVariable("awsregion")

	textractSession = textract.New(session.Must(session.NewSession(&aws.Config{
		Region:      aws.String(awsregion),
		Credentials: credentials.NewStaticCredentials(awsaccesskey, awssecret, awstoken),
	})))
}

func main() {
	filename := os.Args[1]
	print("processing file name is " + filename)
	byteArray, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Failed to read file: %s", err)
	}

	extractdocumentcontent(byteArray, filename)
}

func extractdocumentcontent(fileinput []byte, sourcefilename string) {
	file := fileinput

	resp, err := textractSession.DetectDocumentText(&textract.DetectDocumentTextInput{
		Document: &textract.Document{
			Bytes: file,
		},
	})
	if err != nil {
		panic(err)
	}

	//fmt.Println(resp)
	writejsonresponsetofile(*resp, sourcefilename)

	for i := 1; i < len(resp.Blocks); i++ {
		if *resp.Blocks[i].BlockType == "LINE" {
			fmt.Println(*resp.Blocks[i].Text)
		}
	}
}

func viperEnvVariable(key string) string {

	// SetConfigFile explicitly defines the path, name and extension of the config file.
	// Viper will use this and not check any of the config paths.
	// .env - It will search for the .env file in the current directory
	viper.SetConfigFile("./.env")

	// Find and read the config file
	err := viper.ReadInConfig()

	if err != nil {
		log.Fatalf("Error while reading config file %s", err)
	}

	// viper.Get() returns an empty interface{}
	// to get the underlying type of the key,
	// we have to do the type assertion, we know the underlying value is string
	// if we type assert to other type it will throw an error
	value, ok := viper.Get(key).(string)

	// If the type is a string then ok will be true
	// ok will make sure the program not break
	if !ok {
		log.Fatalf("Invalid type assertion")
	}

	return value
}

func writejsonresponsetofile(resp textract.DetectDocumentTextOutput, outputfilename string) {
	jsonfilename := strings.Replace(outputfilename, ".pdf", ".json", 1)
	jsonfile, fileerr := os.Create(jsonfilename)
	if fileerr != nil {
		log.Fatalf("Failed to write file: %s", fileerr)
	}
	defer jsonfile.Close()
	// Create a JSON encoder
	encoder := json.NewEncoder(jsonfile)
	encoder.SetIndent("", "  ") // Pretty-print the JSON with indentation

	// Encode data to the file
	encodeerr := encoder.Encode(resp)
	if encodeerr != nil {
		log.Fatalf("Failed to write file: %s", encodeerr)
	}
}
