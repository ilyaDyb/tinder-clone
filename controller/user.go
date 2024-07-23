package controller

import (
	// "log"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/ilyaDyb/go_rest_api/config"
	"github.com/ilyaDyb/go_rest_api/models"
	_ "github.com/ilyaDyb/go_rest_api/utils"
)

// @Summary  User profile
// @Accept   json
// @Produce  json
// @Param Authorization header string true "With the Bearer started"
// @Param username path string false "Username"
// @Success  200 {object} utils.ModelResponse
// @Failure  403 {object} utils.ErrorResponse
// @Failure  404 {object} utils.ErrorResponse
// @Router   /u/profile [get]
// @Router   /u/profile/{username} [get]
func ProfileController(c *gin.Context) {
	username := c.Param("username")
	username = username[1:]
	if username == "" || username == "/" {
		usernameFromToken, exists := c.Get("username")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden!"})
			return
		}
		username = usernameFromToken.(string)
	}
	var user models.User
	if err := config.DB.Preload("Photo").Where("username = ?", username).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

type ChangeProfileInput struct {
	Firstname string                `form:"firstname"`
	Lastname  string                `form:"lastname"`
	Age       uint8                 `form:"age"`
	Country   string                `form:"country"`
	Bio       string                `form:"bio"`
	Hobbies   string                `form:"hobbies"`
	Photo     *multipart.FileHeader `form:"photo"`
}

// EditProfileController edits user profile
// @Summary Edit user profile
// @Description Edit user profile details including uploading a profile photo
// @Accept multipart/form-data
// @Produce json
// @Param Authorization header string true "With the Bearer started"
// @Param firstname formData string false "First Name"
// @Param lastname formData string false "Last Name"
// @Param age formData uint8 false "Age"
// @Param country formData string false "Country"
// @Param bio formData string false "Bio"
// @Param hobbies formData string false "Hobbies"
// @Param photo formData file false "Profile Photo"
// @Success 200 {object} utils.MessageResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 403 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /u/profile [put]
func EditProfileController(c *gin.Context) {
	currentUsername := c.MustGet("username").(string)
	var input ChangeProfileInput
	var user models.User

	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := config.DB.Table("users").Where("username = ?", currentUsername).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	fmt.Printf("Input: %+v\n", input)

	// Update user's profile fields
	user.Firstname = input.Firstname
	user.Lastname = input.Lastname
	user.Age = input.Age
	user.Country = input.Country
	user.Bio = input.Bio
	user.Hobbies = input.Hobbies

	file, err := c.FormFile("photo")
	if err == nil {
		if _, err := os.Stat(config.UserPhotoPath); os.IsNotExist(err) {
			os.MkdirAll(config.UserPhotoPath, os.ModePerm)
		}

		filePath := filepath.Join(config.UserPhotoPath, fmt.Sprintf("%d_%s", user.Id, file.Filename))
		log.Println(filePath)
		if err := c.SaveUploadedFile(file, filePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to save file"})
			return
		}
		photo := models.Photo{
			UserID: user.ID,
			URL:    filePath,
		}
		config.DB.Create(&photo)
		user.Photo = append(user.Photo, photo)
	}

	if err := config.DB.Model(&user).Updates(user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to update user profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
}

// SetAsPriview change user preview photo
// @Summary Set as preview
// @Accept json
// @Produce json
// @Param Authorization header string true "With the Bearer started"
// @Param photo_id path uint true "Id for photo which you want to set as privew"
// @Success 200 {object} utils.MessageResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /u/set-as-preview/{photo_id} [put]
func SetAsPriview(c *gin.Context) {
	username := c.MustGet("username").(string)
	photoId := c.Param("photo_id")

	var user models.User
	if err := config.DB.Where("username = ?", username).First(&user).Error; err != nil { 
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	tx := config.DB.Begin()
	
	if err := tx.Model(&models.Photo{}).Where("id = ? AND user_id = ?", photoId, user.Id).Update("is_preview", true).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	if err := tx.Model(&models.Photo{}).Where("user_id = ? AND id != ?", user.Id, photoId).Update("is_preview", false).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	tx.Commit()

	c.JSON(http.StatusOK, gin.H{"message": "Changed preview photo"})
}

type LocationInput struct {
	Lat float32 `json:"lat"`
	Lon float32 `json:"lon"`
}

// @Summary      Save location
// @Accept       json
// @Produce      json
// @Param Authorization header string true "With the Bearer started"
// @Param        LocationInput  body      LocationInput  true  "Location with lat, lon "
// @Success      200         {object}  utils.MessageResponse
// @Failure      500         {object}  utils.ErrorResponse
// @Router       /u/save-location [post]
func SaveLocation(c *gin.Context) {
	username := c.MustGet("username").(string)
	var input LocationInput
	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	var user models.User
	if err := config.DB.Where("username = ?", username).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	
	tx := config.DB.Begin()
	if err := config.DB.Model(&user).UpdateColumn("lat", input.Lat).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save location"})
		return
	}
	if err := config.DB.Model(&user).UpdateColumn("lon", input.Lon).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save location"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Location saved successfully"})
}