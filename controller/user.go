package controller

import (
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ilyaDyb/go_rest_api/api"
	"github.com/ilyaDyb/go_rest_api/config"
	"github.com/ilyaDyb/go_rest_api/models"
	"github.com/ilyaDyb/go_rest_api/service"
	"github.com/ilyaDyb/go_rest_api/utils"
	"github.com/rosberry/go-pagination"
)

type UserController struct {
	userService *service.UserService
}

func NewUserController(userService *service.UserService) *UserController {
	return &UserController{userService: userService}
}

// @Summary  User profile
// @Tags user
// @Accept   json
// @Produce  json
// @Param Authorization header string true "With the Bearer started"
// @Param username path string false "Username"
// @Success  200 {object} utils.ModelResponse
// @Failure  403 {object} utils.ErrorResponse
// @Failure  404 {object} utils.ErrorResponse
// @Router   /u/profile [get]
// @Router   /u/profile/{username} [get]
func (ctrl *UserController) ProfileController(c *gin.Context) {
	username := c.Param("username")
	username = username[1:]
	if username == "" || username == "/" {
		usernameFromToken := c.MustGet("username").(string)
		username = usernameFromToken
	}
	user, err := ctrl.userService.GetUserByUsername(username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	log.Println(user)
	c.JSON(http.StatusOK, gin.H{
		"user":         user,
		"count_photos": len(user.Photo),
	})
}

type ChangeProfileInput struct {
	Firstname string                `form:"firstname" validate:"max=20"`
	Lastname  string                `form:"lastname" validate:"max=20"`
	Age       uint8                 `form:"age" validate:"min=18,max=99"`
	Country   string                `form:"country" validate:"max=30"`
	City      string                `form:"city" validate:"max=30"`
	Bio       string                `form:"bio" validate:"max=500"`
	Hobbies   string                `form:"hobbies" validate:"max=100"`
	Photo     *multipart.FileHeader `form:"photo"`
}

// EditProfileController edits user profile
// @Summary Edit user profile
// @Tags user
// @Description Edit user profile details including uploading a profile photo
// @Accept multipart/form-data
// @Produce json
// @Param Authorization header string true "With the Bearer started"
// @Param firstname formData string false "First Name"
// @Param lastname formData string false "Last Name"
// @Param age formData uint8 false "Age"
// @Param country formData string false "Country"
// @Param city formData string false "City"
// @Param bio formData string false "Bio"
// @Param hobbies formData string false "Hobbies"
// @Param photo formData file false "Profile Photo"
// @Success 200 {object} utils.MessageResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 403 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /u/profile [put]
func (ctrl *UserController) EditProfileController(c *gin.Context) {
	currentUsername := c.MustGet("username").(string)
	var input ChangeProfileInput
	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := utils.ValidateStruct(input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := ctrl.userService.GetUserByUsername(currentUsername)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	user.Firstname = input.Firstname
	user.Lastname = input.Lastname
	user.Age = input.Age
	user.Country = input.Country
	user.City = input.City
	user.Bio = input.Bio
	user.Hobbies = input.Hobbies   
	file, err := c.FormFile("photo")
	if err == nil {
		if _, err := os.Stat(config.UserPhotoPath); os.IsNotExist(err) {
			os.MkdirAll(config.UserPhotoPath, os.ModePerm)
		}

		filePath := filepath.Join(config.UserPhotoPath, fmt.Sprintf("%d_%s", user.ID, file.Filename))
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

	if err := ctrl.userService.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to update user profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
}

// SetAsPriview change user preview photo
// @Summary Set as preview
// @Tags user
// @Accept json
// @Produce json
// @Param Authorization header string true "With the Bearer started"
// @Param photo_id path uint true "Id for photo which you want to set as privew"
// @Success 200 {object} utils.MessageResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /u/set-as-preview/{photo_id} [patch]
func (ctrl *UserController) SetAsPriviewController(c *gin.Context) {
	username := c.MustGet("username").(string)
	photoId, err := strconv.Atoi(c.Param("photo_id"))
	if err != nil {
		c.Status(400)
		return
	}

	user, err := ctrl.userService.GetUserByUsername(username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	err = ctrl.userService.SetPreviewPhoto(user.ID, uint(photoId))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Changed preview photo"})
}

type LocationInput struct {
	Lat float32 `json:"lat"`
	Lon float32 `json:"lon"`
}

// @Summary      Save location
// @Tags user
// @Accept       json
// @Produce      json
// @Param Authorization header string true "With the Bearer started"
// @Param        LocationInput  body      LocationInput  true  "Location with lat, lon "
// @Success      200         {object}  utils.MessageResponse
// @Failure      500         {object}  utils.ErrorResponse
// @Router       /u/save-location [patch]
func (ctrl *UserController) SaveLocationController(c *gin.Context) {
	username := c.MustGet("username").(string)
	var input LocationInput
	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := ctrl.userService.SaveLocation(username, input.Lat, input.Lon)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Location saved successfully"})
}

// @Summary      Save location
// @Tags user
// @Accept       json
// @Produce      json
// @Param Authorization header string true "With the Bearer started"
// @Success      200         {object}  utils.MessageResponse
// @Failure      500         {object}  utils.ErrorResponse
// @Router       /u/set-coordinates [patch]
func (ctrl *UserController) SetCoordinatesController(c *gin.Context) {
	username := c.MustGet("username").(string)

	user, err := ctrl.userService.GetUserByUsername(username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User bot found"})
		return
	}

	country := user.Country
	city := user.City
	place := fmt.Sprintf("%s %s", country, city)
	lat, lon, err := api.GetCoordinates(place)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"lat": lat, "lon": lon})
}

// @Summary      Url for getting users which liked me
// @Tags user
// @Accept       json
// @Produce      json
// @Param Authorization header string true "With the Bearer started"
// @Success      200         {object}  utils.MessageResponse
// @Failure      500         {object}  utils.ErrorResponse
// @Router       /u/liked-by-users [get]
func (ctrl *UserController) LikedByUsersController(c *gin.Context) {
	username := c.MustGet("username").(string)
	user, err := ctrl.userService.GetUserByUsername(username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	usersWhichLikedMe, err := ctrl.userService.GetUsersWhoLikedMe(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, usersWhichLikedMe)
}

// func GetUsersList(userID uint, role string, paginator *pagination.Paginator) []models.User {
// 	var curUser models.User
// 	config.DB.First(&curUser, userID)

// 	var users []models.User

// 	var interactedIDs []uint
// 	config.DB.Model(&models.UserInteraction{}).Where("user_id = ?", userID).Pluck("target_id", &interactedIDs)

// 	ageLower := curUser.Age - 3
// 	ageUpper := curUser.Age + 100
// 	gender := "male"
// 	if curUser.Sex == "male" {
// 		gender = "female"
// 	} else {
// 		gender = "male"
// 	}

// 	q := config.DB.Preload("Photo", "is_preview = ?", true).Model(&models.User{}).
// 		Where("role = ?", role).
// 		Where("id != ?", userID).
// 		Where("age BETWEEN ? and ?", ageLower, ageUpper).
// 		Where("sex = ?", gender).
// 		Where("id NOT IN (?)", interactedIDs)

// 	err := paginator.Find(q, &users)
// 	if err != nil {
// 		log.Println(err)
// 		return nil
// 	}
// 	return users
// }

// @Summary Get profile
// @Tags user
// @Accept json
// @Produce json
// @Param Authorization header string true "With the Bearer started"
// @Router /u/get-profiles [get]
func (ctrl *UserController) GetProfilesController(c *gin.Context) {
	username := c.MustGet("username").(string)
	user, err := ctrl.userService.GetUserByUsername(username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	userID := user.ID
	// if user have subscription then set limit = 100 for example
	paginator, err := pagination.New(pagination.Options{
		GinContext: c,
		DB:         config.DB,
		Model:      &models.User{},
		Limit:      2,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	users, err := ctrl.userService.GetUsersList(userID, "user", paginator)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, utils.UsersListResponse{
		Result:     true,
		Users:      users,
		Pagination: paginator.PageInfo,
	})
}

// 	users := GetUsersList(userID, "user", paginator)

// 	c.JSON(http.StatusOK, usersListResponse{
// 		Result:     true,
// 		Users:      users,
// 		Pagination: paginator.PageInfo,
// 	})
// }

type InputGrade struct {
	TargetID  uint
	InterType string
}

// @Summary to Grade profiles
// @Tags user
// @Accept json
// @Produce json
// @Param Authorization header string true "With the Bearer started"
// @Param InputGrade body InputGrade true "Input for Grade other profile"
// @Router /u/grade [post]
func (ctrl *UserController) GradeProfileController(c *gin.Context) {
	username := c.MustGet("username").(string)
	user, err := ctrl.userService.GetUserByUsername(username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if user.RestrictionEnd.After(time.Now()) && !(user.RestrictionEnd.IsZero()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("you have restriction for interaction expire at %02d-%02d", user.RestrictionEnd.Month(), user.RestrictionEnd.Day())})
		return
	}
	var input InputGrade
	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	InterType := input.InterType
	if InterType != "like" && InterType != "dislike" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Interaction should be 'like' or 'dislike'"})
		return
	}
	targetId := input.TargetID
	var interaction models.UserInteraction
	interaction.TargetID = targetId
	interaction.UserID = user.ID
	interaction.InteractionType = InterType
	if err := config.DB.Create(&interaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	userInteractionsCount, err := ctrl.userService.GetUserInteractionsCount(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Check if the user is subscribed if not
	// if subscriber {limit = 100} else {limit = 10} countOfInteraction%limit == 0 && != 0
	if userInteractionsCount%10 == 0 && userInteractionsCount != 0 {
		RestrictionEnd := time.Now().Add(24 * time.Hour)
		user.RestrictionEnd = RestrictionEnd
		if err := config.DB.Save(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.Status(http.StatusOK)

}
