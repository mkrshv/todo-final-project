package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"test/repository"
	taskservice "test/task-service"

	"github.com/golang-jwt/jwt"
)

type Handler struct {
	RP repository.RepositoryProcesser
}

// Интерфейс для работы с объектом типа обработчик (Handler).
type HandleProcesser interface {
	HandleTask(w http.ResponseWriter, r *http.Request)
	HandleDate(w http.ResponseWriter, r *http.Request)
	GetTasksHandle(w http.ResponseWriter, r *http.Request)
	GetTaskHandle(w http.ResponseWriter, r *http.Request)
	PutTaskHandle(w http.ResponseWriter, r *http.Request)
	DoneTaskeHandle(w http.ResponseWriter, r *http.Request)
	Auth(w http.ResponseWriter, r *http.Request)
	AuthMiddleware(next http.HandlerFunc) http.HandlerFunc
}

func NewHandler() Handler {
	rp, err := repository.NewRepo()
	if err != nil {
		panic(err)
	}
	return Handler{RP: rp}
}

// Обработчик возвращающий следующую даты для выполненной задачи.
func (h Handler) HandleDate(w http.ResponseWriter, r *http.Request) {
	task := new(taskservice.Task)
	task.Date = r.FormValue("date")
	task.Repeat = r.FormValue("repeat")
	now := r.FormValue("now")
	nextDt, err := task.GetNextRepeatDateTest(now)
	if err != nil {
		log.Print(err)
		JsonErr(w, http.StatusBadRequest, err.Error())
		return
	}

	w.Write([]byte(nextDt))
}

// Обработчик, поведение которого зависит от метода в r *http.Request.
func (h Handler) HandleTask(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		h.PostHandle(w, r)
	case "GET":
		h.GetTaskHandle(w, r)
	case "PUT":
		h.PutTaskHandle(w, r)
	case "DELETE":
		h.DeleteTaskeHandle(w, r)
	default:
		return
	}
}

// Вспомогательная функция, посылающая ответ об ошибке в формате JSON.
func JsonErr(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)

	errorResponse := map[string]string{
		"error": message,
	}

	// Сериализуем карту в JSON
	response, err := json.Marshal(errorResponse)
	if err != nil {
		// В случае ошибки сериализации возвращаем простую текстовую ошибку
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Записываем результат в http.ResponseWriter
	w.Write(response)
}

// Вспомогательная функция, посылающая ответ на запрос в формате JSON.
func JsonResponse(w http.ResponseWriter, statusCode int, id string) {
	w.WriteHeader(statusCode)

	resp := map[string]string{
		"id": id,
	}

	// Сериализуем карту в JSON
	response, err := json.Marshal(resp)
	if err != nil {
		// В случае ошибки сериализации возвращаем простую текстовую ошибку
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Записываем результат в http.ResponseWriter
	w.Write(response)
}

// Обработчик размещающий задачу в репозитории, если она  соответствует требованиям.
func (h Handler) PostHandle(w http.ResponseWriter, r *http.Request) {
	var newTask taskservice.Task
	var buf bytes.Buffer

	_, err := buf.ReadFrom(r.Body)

	if err != nil {
		log.Print(err)
		JsonErr(w, http.StatusBadRequest, "1234455N")
		return
	}

	w.Header().Set("Content-type", "application/json")

	if err := json.Unmarshal(buf.Bytes(), &newTask); err != nil {
		log.Print(err)
		JsonErr(w, http.StatusBadRequest, "Ошибка десериализации JSON")
		return
	}

	id, err := h.RP.AddTask(newTask)
	if err != nil {
		log.Print(err)
		JsonErr(w, http.StatusBadRequest, err.Error())
		return
	}

	JsonResponse(w, http.StatusOK, id)
}

// Обработчик возвращающий список из 10 ближайших задач.
func (h Handler) GetTasksHandle(w http.ResponseWriter, r *http.Request) {
	search := r.FormValue("search")

	if search != "" {
		taskSLice, err := h.RP.SearchTask(search)
		if err != nil {
			log.Print(err)
			JsonErr(w, http.StatusBadRequest, err.Error())
			return
		}
		respMap := make(map[string][]taskservice.Task)
		respMap["tasks"] = taskSLice
		resp, err := json.Marshal(respMap)
		if err != nil {
			log.Print(err)
			JsonErr(w, http.StatusBadRequest, err.Error())
			return
		}
		w.Write(resp)
		return
	}

	taskSLice, err := h.RP.GetTaskList()
	if err != nil {
		log.Print(err)
		JsonErr(w, http.StatusBadRequest, err.Error())
	}

	respMap := make(map[string][]taskservice.Task)
	respMap["tasks"] = taskSLice

	resp, err := json.Marshal(respMap)
	if err != nil {
		log.Print(err)
		JsonErr(w, http.StatusInternalServerError, err.Error())
	}
	k := make(map[string][]taskservice.Task)
	err = json.Unmarshal(resp, &k)
	if err != nil {
		log.Print(err)
	}

	w.Write(resp)
}

// Обработчик возвращающий задачу по id.
func (h Handler) GetTaskHandle(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")

	task, err := h.RP.GetTask(id)
	if err != nil {
		log.Print(err)
		JsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := json.Marshal(task)
	if err != nil {
		log.Print(err)
		JsonErr(w, http.StatusBadRequest, err.Error())
		return
	}

	w.Write(resp)
}

// Обработчик обновляющий задачу.
func (h Handler) PutTaskHandle(w http.ResponseWriter, r *http.Request) {

	var buf bytes.Buffer
	task := taskservice.Task{}
	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		log.Print(err)
	}

	err = json.Unmarshal(buf.Bytes(), &task)
	if err != nil {
		log.Print(err)
		JsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	err = h.RP.UpdateTask(task)
	if err != nil {
		log.Print(err)
		JsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	response := struct{}{}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}

// Обработчик выполнения задачи.
func (h Handler) DoneTaskeHandle(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")

	err := h.RP.DoneTask(id)
	if err != nil {
		log.Print(err)
		JsonErr(w, http.StatusBadRequest, "wrong id")
		return
	}

	response := struct{}{}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Обработчик удаления задачи.
func (h Handler) DeleteTaskeHandle(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	err := h.RP.DeleteTask(id)
	if err != nil {
		log.Print(err)
		JsonErr(w, http.StatusBadRequest, "wrong id")
		return
	}
	response := struct{}{}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type AuthPass struct {
	Password string `json:"password"`
}

func (h Handler) Auth(w http.ResponseWriter, r *http.Request) {
	passwd := os.Getenv("TODO_PASSWORD")

	var buf bytes.Buffer
	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		log.Print(err)
	}

	var auth AuthPass
	err = json.Unmarshal(buf.Bytes(), &auth)
	if err != nil {
		log.Print(err)
	}

	if auth.Password == passwd {
		jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sum": sha256.Sum256([]byte(auth.Password)),
		})
		tokenString, err := jwtToken.SignedString([]byte(passwd))
		if err != nil {
			log.Print(err)
			JsonErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		fmt.Println(tokenString)
		w.WriteHeader(http.StatusOK)

		resp := map[string]string{
			"token": tokenString,
		}

		response, err := json.Marshal(resp)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		// Записываем результат в http.ResponseWriter
		w.Write(response)
		return
	}

	JsonErr(w, http.StatusUnauthorized, "wrong password")
}

func (h Handler) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// смотрим наличие пароля
		pass := os.Getenv("TODO_PASSWORD")
		if len(pass) > 0 {
			var jwtToken string // JWT-токен из куки
			// получаем куку
			cookie, err := r.Cookie("token")
			if err == nil {
				jwtToken = cookie.Value
			}
			var valid bool

			token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("неожиданный метод подписи: %v", token.Header["alg"])
				}
				return []byte(os.Getenv("TODO_PASSWORD")), nil
			})
			if err == nil && token.Valid {
				valid = true
			}

			if !valid {
				// возвращаем ошибку авторизации 401
				http.Error(w, "Authentification required", http.StatusUnauthorized)
				return
			}
		}
		next(w, r)
	})
}
