package task

import (
	"context"
	"myapp/dto"
	"myapp/middleware"
	"myapp/model"
	"myapp/services"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func CreateTaskController(router *gin.Engine, firestoreClient *firestore.Client) {

	router.POST("/task", middleware.AccessTokenMiddleware(), func(c *gin.Context) {
		Createtask(c, firestoreClient)
	})
}

func Createtask(c *gin.Context, firestoreClient *firestore.Client) {
	userId := c.MustGet("userId").(string)
	var taskReq dto.CreateTaskRequest
	if err := c.ShouldBindJSON(&taskReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// ค้นหาผู้ใช้จากฐานข้อมูล
	ctx := context.Background()
	_, err := services.GetUserDataByUserid(ctx, firestoreClient, userId)
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}

	taskid := uuid.New().String()

	newtask := model.Tasks{
		TaskID:      taskid,
		BoardID:     taskReq.BoardID,
		TaskName:    taskReq.TaskName,
		Description: taskReq.Description,
		Status:      taskReq.Status,
		Priority:    taskReq.Priority,
		CreatedBy:   userId,
		UpdatedAt:   time.Now(),
	}

	// บันทึก Task ลง collection Tasks ด้วย document ID = taskid
	_, err = firestoreClient.Collection("Tasks").Doc(taskid).Set(ctx, newtask)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to create task"})
		return
	}

	// ตรวจสอบว่ามี Reminder หรือไม่
	if taskReq.Reminder != nil {
		notidicationid := uuid.New().String()

		// แปลง due_date จาก string เป็น *time.Time (pointer)
		var dueDate *time.Time
		if taskReq.Reminder.DueDate != "" {
			// สมมติว่า format เป็น "2006-01-02T15:04:05Z07:00" หรือตาม format ที่คุณใช้
			parsedDate, err := time.Parse(time.RFC3339, taskReq.Reminder.DueDate)
			if err != nil {
				// ถ้า parse ไม่ได้ ให้ใช้ format อื่น หรือ handle error ตามต้องการ
				c.JSON(400, gin.H{"error": "Invalid due_date format"})
				return
			}
			dueDate = &parsedDate // ใช้ address ของ parsedDate
		}

		// แปลง before_due_date จาก string เป็น *time.Time (ถ้ามี)
		var beforeDueDate *time.Time
		if taskReq.Reminder.BeforeDueDate != nil && *taskReq.Reminder.BeforeDueDate != "" {
			parsedBeforeDueDate, err := time.Parse(time.RFC3339, *taskReq.Reminder.BeforeDueDate)
			if err != nil {
				c.JSON(400, gin.H{"error": "Invalid before_due_date format"})
				return
			}
			beforeDueDate = &parsedBeforeDueDate
		}

		// แปลง recurring_pattern เป็น *string
		var recurringPattern *string
		if taskReq.Reminder.RecurringPattern != "" {
			recurringPattern = &taskReq.Reminder.RecurringPattern
		}

		newnotification := model.Notification{
			NotificationID:   notidicationid,
			TaskID:           taskid,
			DueDate:          dueDate,
			BeforeDueDate:    beforeDueDate,
			RecurringPattern: recurringPattern,
			Send:             "0", // default value สำหรับ Send status
			Updatedat:        time.Now(),
		}

		// บันทึก Notification ลง collection Notifications ด้วย document ID = notidicationid
		_, err = firestoreClient.Collection("NotificationTasks").Doc(notidicationid).Set(ctx, newnotification)
		if err != nil {
			// ถ้าบันทึก notification ไม่สำเร็จ อาจต้อง rollback task ที่สร้างไปแล้ว
			// ลบ task ที่สร้างไปแล้ว
			firestoreClient.Collection("Tasks").Doc(taskid).Delete(ctx)
			c.JSON(500, gin.H{"error": "Failed to create notification"})
			return
		}
	}

	response := gin.H{
		"message": "Task created successfully",
		"taskID":  taskid,
	}

	c.JSON(http.StatusCreated, response)
}
