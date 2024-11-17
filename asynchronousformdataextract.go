package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
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
	// Call the AnalyzeDocument API with the "FORMS" feature type
	input := &textract.AnalyzeDocumentInput{
		Document: &textract.Document{
			S3Object: &textract.S3Object{
				Bucket: aws.String(viperEnvVariable(("awss3bucket"))),
				Name:   aws.String(filename),
			},
		},
		FeatureTypes: []*string{
			aws.String("FORMS"),
		},
	}

	result, err := textractSession.AnalyzeDocumentWithContext(context.Background(), input)
	if err != nil {
		log.Fatalf("Failed to analyze document: %s", err)
	}

	// Parse the response to extract key-value pairs
	keyValuePairs := extractKeyValuePairs(result)

	// Extract keys and sort them
	keys := make([]string, 0, len(keyValuePairs))
	for key := range keyValuePairs {
		keys = append(keys, key)
	}
	sort.Strings(keys) // Sort keys in alphabetical order

	// Print the extracted key-value pairs
	fmt.Println("Extracted Key-Value Pairs:")
	for _, key := range keys {
		fmt.Printf("%s: %s\n", key, keyValuePairs[key])
	}
}

// extractKeyValuePairs parses the Textract response for key-value pairs
func extractKeyValuePairs(result *textract.AnalyzeDocumentOutput) map[string]string {
	keyMap := make(map[string]*textract.Block)
	valueMap := make(map[string]*textract.Block)
	blockMap := make(map[string]*textract.Block)

	// Organize blocks into maps
	for _, block := range result.Blocks {
		blockMap[*block.Id] = block
		if *block.BlockType == "KEY_VALUE_SET" {
			if contains(block.EntityTypes, "KEY") {
				keyMap[*block.Id] = block
			} else if contains(block.EntityTypes, "VALUE") {
				valueMap[*block.Id] = block
			}
		}
	}

	// Extract key-value pairs
	keyValuePairs := make(map[string]string)
	for _, keyBlock := range keyMap {
		// Find the value block associated with the key block
		for _, rel := range keyBlock.Relationships {
			if *rel.Type == "VALUE" {
				for _, valueID := range rel.Ids {
					valueBlock := valueMap[*valueID]
					key := getText(keyBlock, blockMap)
					value := getText(valueBlock, blockMap)
					keyValuePairs[key] = value
				}
			}
		}
	}

	return keyValuePairs
}

// getText retrieves the text from a block and its child blocks
func getText(block *textract.Block, blockMap map[string]*textract.Block) string {
	text := strings.Builder{}
	for _, rel := range block.Relationships {
		if *rel.Type == "CHILD" {
			for _, childID := range rel.Ids {
				childBlock := blockMap[*childID]
				if childBlock.Text != nil {
					text.WriteString(*childBlock.Text)
					text.WriteString(" ")
				}
			}
		}
	}
	return strings.TrimSpace(text.String())
}

// contains checks if a string slice contains a specific string
func contains(slice []*string, str string) bool {
	for _, item := range slice {
		if *item == str {
			return true
		}
	}
	return false
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
