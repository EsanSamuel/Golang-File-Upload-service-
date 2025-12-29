package main

import (
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type FileDTO struct {
	FileName   string
	FileSize   int64
	FileHeader map[string][]string
}

func main() {
	err := godotenv.Load(".env")

	if err != nil {
		log.Println("Warning: unable to find .env file")
	}
	router := gin.Default()
	router.POST("/upload", fileUploadHandler)

	router.Run(":8080")

	fmt.Println("Server is running at port :8080")
}

func fileUploadHandler(c *gin.Context) {
	file, err := c.FormFile("file")

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Failed to get file"})
		return
	}

	var fileDTO FileDTO

	fileDTO.update(file.Filename, file.Header, file.Size)
	if ok := returnFileDTO(file.Filename, file.Header, file.Size); ok != nil {
		fmt.Println("File details:", ok)
		c.JSON(http.StatusOK, gin.H{"FileDto": ok})
	}

	src, err := file.Open()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer src.Close()
	url, err := uploadToS3(src, fileDTO.FileName)

	if url != "" {
		c.JSON(http.StatusOK, gin.H{"message": "File uploaded to S3", "URL": url})
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong  uploading to s3", "details": err.Error()})
		return
	}
	fmt.Println("File url:", url)

}

func (f *FileDTO) update(filename string, fileHeader map[string][]string, filesize int64) {
	f.FileName = filename
	f.FileHeader = fileHeader
	f.FileSize = filesize
}

func returnFileDTO(filename string, fileHeader map[string][]string, filesize int64) *FileDTO {
	return &FileDTO{
		FileName:   filename,
		FileHeader: fileHeader,
		FileSize:   filesize,
	}
}

func uploadToS3(file multipart.File, filename string) (string, error) {
	defer file.Close()

	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	bucketName := os.Getenv("AWS_BUCKET_NAME")

	sess, err := session.NewSession(&aws.Config{
		Region:   aws.String("us-west-1"),
		Endpoint: aws.String("https://t3.storage.dev"),
		Credentials: credentials.NewStaticCredentials(
			accessKey, secretKey, "",
		),
		S3ForcePathStyle: aws.Bool(true),
	})
	if err != nil {
		return "", err
	}

	s3Client := s3.New(sess)
	_, err = s3Client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(filename),
		Body:   file,
		ACL:    aws.String("public-read"),
	})

	url := "https://file-uploads.t3.storage.dev/" + filename

	return url, err
}
