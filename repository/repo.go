package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	taskservice "test/task-service"
	"time"

	_ "modernc.org/sqlite"
)

const (
	ErrNoId     = "Не указан идентификатор"
	ErrNotFound = "Задача не найдена"
	DateFormat  = "20060102"
)

type Repository struct {
	Repo *sql.DB
}

// Интерфейс для работы с репозиторием
type RepositoryProcesser interface {
	AddTask(task taskservice.Task) (string, error)
	GetTaskList() ([]taskservice.Task, error)
	GetTask(id string) (taskservice.Task, error)
	UpdateTask(task taskservice.Task) error
	DoneTask(id string) error
	DeleteTask(id string) error
	SearchTask(search string) ([]taskservice.Task, error)
}

// Создает (в случае необходимости) и открывает доступ к БД. Возвращает ссылку на объект типа Repository.
func NewRepo() (*Repository, error) {

	dbFile := os.Getenv("TODO_DFILE")
	if dbFile == "" {
		dbFile = dbCheck()
	}

	fmt.Println(dbFile)

	repo := Repository{}

	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		panic(err)
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS scheduler (id INTEGER PRIMARY KEY AUTOINCREMENT, date TEXT,  title TEXT, comment TEXT, repeat TEXT)")
	if err != nil {
		panic(err)
	}
	repo.Repo = db
	if err = db.Ping(); err != nil {
		panic(err)
	}
	return &repo, nil
}

// Вспомогательная функция, проверяющая наличие БД в месте запуска программы.
func dbCheck() string {
	appPath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(appPath)
	dbFile := filepath.Join(filepath.Dir(appPath), "scheduler.db")
	_, err = os.Stat(dbFile)

	if err != nil {
		os.Create("scheduler.db")
	}

	return dbFile
}

// Добавляет задачу в БД.
func (repo *Repository) AddTask(task taskservice.Task) (string, error) {
	nextDate, err := task.GetNextRepeatDate()
	if err != nil {
		return "", err
	}

	if task.Title == "" {
		return "", errors.New("no title")
	}

	if task.Date == "" {
		task.Date = time.Now().Format(DateFormat) // Присваиваем текущую дату
	}

	date, err := time.Parse(DateFormat, task.Date)
	if err != nil {
		return "", err
	}
	// Здесь можно проверять, если дата уже прошла
	if date.Before(time.Now().Truncate(24 * time.Hour)) {
		switch task.Repeat {
		case "":
			date = time.Now()
		default:
			nextDateParsed, err := time.Parse(DateFormat, nextDate)
			if err != nil {
				return "", err
			}
			date = nextDateParsed // Иначе используем следующую дату
		}
	}
	task.Date = date.Format(DateFormat) // Устанавливаем отформатированную дату

	res, err := repo.Repo.Exec("INSERT INTO scheduler (date, title, comment, repeat) VALUES (:date, :title, :comment, :repeat)",
		sql.Named("date", task.Date),
		sql.Named("title", task.Title),
		sql.Named("comment", task.Comment),
		sql.Named("repeat", task.Repeat))
	if err != nil {
		return "", err
	}

	id, _ := res.LastInsertId()
	strid := strconv.Itoa(int(id))
	return strid, nil
}

// Возвращает список (срез) 10 ближайших по дате задач.
func (repo *Repository) GetTaskList() ([]taskservice.Task, error) {
	result := []taskservice.Task{}

	rows, err := repo.Repo.Query("SELECT * FROM scheduler ORDER BY date LIMIT 10")

	if err != nil {
		return result, err
	}

	for rows.Next() {
		task := taskservice.Task{}
		err := rows.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
		if err != nil {
			fmt.Println(err)
			return result, err
		}
		result = append(result, task)
	}

	if err := rows.Err(); err != nil {
		fmt.Println(err)
		return result, err
	}

	return result, nil
}

// Возвращает задачу по id в виде структуры типа Task.
func (repo *Repository) GetTask(id string) (taskservice.Task, error) {
	task := taskservice.Task{}
	if id == "" {
		return task, fmt.Errorf(ErrNoId)
	}
	row := repo.Repo.QueryRow("SELECT id, date, title, comment, repeat FROM scheduler WHERE id = :id", sql.Named("id", id))
	err := row.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err != nil {
		return task, fmt.Errorf(ErrNotFound)
	}
	return task, nil
}

// Обновляет задачу, переданную в запросе.
func (repo *Repository) UpdateTask(task taskservice.Task) error {
	_, err := task.GetNextRepeatDate()
	if err != nil {
		return err
	}

	if _, err = time.Parse(DateFormat, task.Date); err != nil {
		return errors.New("wrong date")
	}

	if task.Title == "" {
		fmt.Println(errors.New("no id"))
		return errors.New("no title")
	}

	if task.ID == "" {
		fmt.Println(errors.New("no id"))
		return errors.New("no id")
	}
	row, err := repo.Repo.Exec("UPDATE scheduler SET date = :date, title = :title, comment = :comment, repeat = :repeat WHERE id = :id",
		sql.Named("date", task.Date),
		sql.Named("title", task.Title),
		sql.Named("comment", task.Comment),
		sql.Named("repeat", task.Repeat),
		sql.Named("id", task.ID))
	if err != nil {
		fmt.Println(err)
		return err
	}
	ra, err := row.RowsAffected()
	if err != nil {
		return err
	}
	if ra != 1 {
		return errors.New("no rows affected")
	}
	return nil
}

// Механизм выполнения задачи: если поле repeat пустое - удаляет задачу,
// в противном случае обновляет дату имеющейся задачи с тем же id.
func (repo *Repository) DoneTask(id string) error {
	task, err := repo.GetTask(id)
	if err != nil {
		return err
	}

	if task.Repeat == "" {
		repo.DeleteTask(id)
		return nil
	}

	nextDate, err := task.GetNextRepeatDate()
	if err != nil {
		return err
	}

	task.Date = nextDate
	err = repo.UpdateTask(task)

	if err != nil {
		return err
	}

	return nil
}

// Удаляет задачу с заданным ID.
func (repo *Repository) DeleteTask(id string) error {
	row, err := repo.Repo.Exec("DELETE FROM scheduler WHERE id=:id", sql.Named("id", id))
	if err != nil {
		return err
	}
	ra, err := row.RowsAffected()
	if err != nil {
		return err
	}
	if ra != 1 {
		return errors.New("no rows affected")
	}
	return nil
}

func (repo *Repository) SearchTask(search string) ([]taskservice.Task, error) {
	res := []taskservice.Task{}
	if date, err := time.Parse("02.01.2006", search); err == nil {
		rows, err := repo.Repo.Query("SELECT * FROM scheduler WHERE date = :date", sql.Named("date", date.Format(DateFormat)))
		if err != nil {
			return res, err
		}

		for rows.Next() {
			task := taskservice.Task{}
			err := rows.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
			if err != nil {
				return res, err
			}
			res = append(res, task)
		}

		if err := rows.Err(); err != nil {
			return res, err
		}

		return res, nil
	}
	rows, err := repo.Repo.Query("SELECT * FROM scheduler WHERE title LIKE :search OR comment LIKE :search", sql.Named("search", "%"+search+"%"))
	if err != nil {
		return res, err
	}

	for rows.Next() {
		task := taskservice.Task{}
		err := rows.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
		if err != nil {
			return res, err
		}
		res = append(res, task)
	}

	if err := rows.Err(); err != nil {
		return res, err
	}

	return res, nil
}
