package task

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Структура единицы репозитория - Task.
type Task struct {
	ID      string `json:"id"`
	Date    string `json:"date"`
	Title   string `json:"title,omitempty"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

type TaskHandler interface {
	GetNextRepeatDate() (string, error)
}

const DateFormat = "20060102"

// Возвращает новую дату для задачи в зависимости от значения, указанного в поле repeat.
func (t *Task) GetNextRepeatDate() (string, error) {
	if t.Date == "" {
		t.Date = time.Now().Format(DateFormat)
	}
	switch {
	case strings.HasPrefix(t.Repeat, "d "):
		daysStr := strings.TrimPrefix(t.Repeat, "d ")
		daysNum, err := strconv.Atoi(daysStr)
		if err != nil {
			return "", fmt.Errorf("неверный формат: %s ; %v", t.Repeat, err)
		}

		if daysNum >= 400 {
			return "", fmt.Errorf("перенос задачи на 400 и более дней: %s;", t.Repeat)
		}

		taskDate, err := time.Parse(DateFormat, t.Date)

		if err != nil {
			return "", fmt.Errorf("ошибка при считывании даты: %s", t.Date)
		}

		taskDate = taskDate.AddDate(0, 0, daysNum) // для учета пограничных вариантов

		for taskDate.Before(time.Now()) {
			taskDate = taskDate.AddDate(0, 0, daysNum)
		}

		return taskDate.Format(DateFormat), nil

	case t.Repeat == "y":
		taskDate, err := time.Parse(DateFormat, t.Date)
		if err != nil {
			return "", fmt.Errorf("ошибка при считывании даты: %s", t.Date)
		}

		taskDate = taskDate.AddDate(1, 0, 0) // для учета пограничных вариантов
		for taskDate.Before(time.Now()) {
			taskDate = taskDate.AddDate(1, 0, 0)
		}
		return taskDate.Format(DateFormat), nil

	case strings.HasPrefix(t.Repeat, "w "):
		weekdaysStr := strings.Split(strings.TrimPrefix(t.Repeat, "w "), ",")
		weekdaysInt := make([]int, len(weekdaysStr))
		for i := range weekdaysStr {
			num, err := strconv.Atoi(weekdaysStr[i])
			if err != nil {
				return "", fmt.Errorf("неверный формат: %s ; %v", t.Repeat, err)
			}
			if num > 7 || num < 0 {
				return "", fmt.Errorf("неверный формат: %s ; %v", t.Repeat, err)
			}
			if num == 7 {
				num = 0
			}
			weekdaysInt[i] = num
		}
		taskDate, err := time.Parse(DateFormat, t.Date)
		if err != nil {
			return "", fmt.Errorf("ошибка при считывании даты: %s", t.Date)
		}

		if !taskDate.After(time.Now()) {
			taskDate = time.Now().AddDate(0, 0, 1)
		}

		found := false
		for {
			for _, day := range weekdaysInt {

				if int(taskDate.Weekday()) == day {
					found = true
					break
				}

			}
			if found {
				break
			}
			taskDate = taskDate.AddDate(0, 0, 1)
		}
		return taskDate.Format(DateFormat), nil

	case strings.HasPrefix(t.Repeat, "m "):
		splitted := strings.Split(t.Repeat, " ")
		fmt.Println(splitted)
		if len(splitted) > 3 || len(splitted) < 2 {
			return "", fmt.Errorf("неверный формат: %s;", t.Repeat)
		}

		daysStr := strings.Split(splitted[1], ",")
		var daysNum []int

		for i := range daysStr {
			dayNum, err := strconv.Atoi(daysStr[i])
			if err != nil || dayNum > 31 || dayNum == 0 || dayNum < -2 {
				return "", fmt.Errorf("неверный формат: %s;", t.Repeat)
			}

			daysNum = append(daysNum, dayNum)
		}

		sort.Slice(daysNum, func(i, j int) bool {
			return daysNum[i] < daysNum[j]
		})

		taskDate, err := time.Parse(DateFormat, t.Date)
		if err != nil {
			return "", fmt.Errorf("ошибка при считывании даты: %s", t.Date)
		}

		if !taskDate.After(time.Now()) {
			taskDate = time.Now().AddDate(0, 0, 1)
		}
		found := false

		taskDate = checkFirstMonth(daysNum, taskDate)

		if len(splitted) == 3 {

			monthsStr := strings.Split(splitted[2], ",")
			var months []int

			for i := range monthsStr {
				mthNum, err := strconv.Atoi(monthsStr[i])
				if err != nil || mthNum > 12 || mthNum < 1 {
					return "", fmt.Errorf("неверный формат: %s;", t.Repeat)
				}

				months = append(months, mthNum)
			}

			sort.Slice(months, func(i, j int) bool {
				return months[i] < months[j]
			})

			for {
				for _, v := range months {
					if int(taskDate.Month()) == v {
						found = true
						break
					}
				}

				if found {
					break
				}

				taskDate = taskDate.AddDate(0, 1, 0)
			}

		}

		found = false
		for {
			for _, v := range daysNum {
				if v < 0 {
					v = v + 1
					lastDay := time.Date(taskDate.Year(), taskDate.Month()+1, 0, 0, 0, 0, 0, time.UTC)
					v = lastDay.AddDate(0, 0, v).Day()
				}

				if taskDate.Day() == v {
					found = true
					break
				}
			}

			if found {
				break
			}

			taskDate = taskDate.AddDate(0, 0, 1)
		}
		return taskDate.Format(DateFormat), nil
	case t.Repeat == "":
		return "", nil
	default:
		return "", fmt.Errorf("неверный формат поля 't.Repeat': %s", t.Repeat)
	}
}

// Вспомогательная функция, проверяющая есть ли в месяце дни, соответствующие указателю repeat.
func checkFirstMonth(daysNum []int, taskDate time.Time) time.Time {
	startMonth := taskDate.Month()
	found := false
	for {
		for _, v := range daysNum {
			if v < 0 {
				v = v + 1
				lastDay := time.Date(taskDate.Year(), taskDate.Month()+1, 0, 0, 0, 0, 0, time.UTC)
				v = lastDay.AddDate(0, 0, v).Day()
			}

			if taskDate.Day() == v {
				found = true
				break
			}
		}

		if found {
			break
		}

		taskDate = taskDate.AddDate(0, 0, 1)
		if taskDate.Month() != startMonth {
			return taskDate
		}
	}
	return taskDate
}

// Возвращает новую дату для задачи в зависимости от значения, указанного в поле repeat.
// Сделана для прохождения тестов. Логика не отличается от используемой в программе функции.
func (t *Task) GetNextRepeatDateTest(now string) (string, error) {
	nowTime, err := time.Parse(DateFormat, now)
	if err != nil {
		return "", err
	}
	switch {
	case strings.HasPrefix(t.Repeat, "d "):
		daysStr := strings.TrimPrefix(t.Repeat, "d ")
		daysNum, err := strconv.Atoi(daysStr)
		if err != nil {
			return "", fmt.Errorf("неверный формат: %s ; %v", t.Repeat, err)
		}

		if daysNum >= 400 {
			return "", fmt.Errorf("перенос задачи на 400 и более дней: %s;", t.Repeat)
		}

		taskDate, err := time.Parse(DateFormat, t.Date)

		if err != nil {
			return "", fmt.Errorf("ошибка при считывании даты: %s", t.Date)
		}

		taskDate = taskDate.AddDate(0, 0, daysNum) // для учета пограничных вариантов

		for taskDate.Before(nowTime) {
			taskDate = taskDate.AddDate(0, 0, daysNum)
		}

		return taskDate.Format(DateFormat), nil

	case t.Repeat == "y":
		taskDate, err := time.Parse(DateFormat, t.Date)
		if err != nil {
			return "", fmt.Errorf("ошибка при считывании даты: %s", t.Date)
		}

		taskDate = taskDate.AddDate(1, 0, 0) // для учета пограничных вариантов
		for taskDate.Before(nowTime) {
			taskDate = taskDate.AddDate(1, 0, 0)
		}
		return taskDate.Format(DateFormat), nil

	case strings.HasPrefix(t.Repeat, "w "):
		weekdaysStr := strings.Split(strings.TrimPrefix(t.Repeat, "w "), ",")
		weekdaysInt := make([]int, len(weekdaysStr))
		for i := range weekdaysStr {
			num, err := strconv.Atoi(weekdaysStr[i])
			if err != nil {
				return "", fmt.Errorf("неверный формат: %s ; %v", t.Repeat, err)
			}
			if num > 7 || num < 0 {
				return "", fmt.Errorf("неверный формат: %s ; %v", t.Repeat, err)
			}
			if num == 7 {
				num = 0
			}
			weekdaysInt[i] = num
		}
		taskDate, err := time.Parse(DateFormat, t.Date)
		if err != nil {
			return "", fmt.Errorf("ошибка при считывании даты: %s", t.Date)
		}

		if !taskDate.After(nowTime) {
			taskDate = nowTime.AddDate(0, 0, 1)
		}

		found := false
		for {
			for _, day := range weekdaysInt {

				if int(taskDate.Weekday()) == day {
					found = true
					break
				}

			}
			if found {
				break
			}
			taskDate = taskDate.AddDate(0, 0, 1)
		}
		return taskDate.Format(DateFormat), nil

	case strings.HasPrefix(t.Repeat, "m "):
		splitted := strings.Split(t.Repeat, " ")
		fmt.Println(splitted)
		if len(splitted) > 3 || len(splitted) < 2 {
			return "", fmt.Errorf("неверный формат: %s;", t.Repeat)
		}

		daysStr := strings.Split(splitted[1], ",")
		var daysNum []int

		for i := range daysStr {
			dayNum, err := strconv.Atoi(daysStr[i])
			if err != nil || dayNum > 31 || dayNum == 0 || dayNum < -2 {
				return "", fmt.Errorf("неверный формат: %s;", t.Repeat)
			}

			daysNum = append(daysNum, dayNum)
		}

		sort.Slice(daysNum, func(i, j int) bool {
			return daysNum[i] < daysNum[j]
		})

		taskDate, err := time.Parse(DateFormat, t.Date)
		if err != nil {
			return "", fmt.Errorf("ошибка при считывании даты: %s", t.Date)
		}

		if !taskDate.After(nowTime) {
			taskDate = nowTime.AddDate(0, 0, 1)
		}
		found := false

		taskDate = checkFirstMonth(daysNum, taskDate)

		if len(splitted) == 3 {

			monthsStr := strings.Split(splitted[2], ",")
			var months []int

			for i := range monthsStr {
				mthNum, err := strconv.Atoi(monthsStr[i])
				if err != nil || mthNum > 12 || mthNum < 1 {
					return "", fmt.Errorf("неверный формат: %s;", t.Repeat)
				}

				months = append(months, mthNum)
			}

			sort.Slice(months, func(i, j int) bool {
				return months[i] < months[j]
			})

			for {
				for _, v := range months {
					if int(taskDate.Month()) == v {
						found = true
						break
					}
				}

				if found {
					break
				}

				taskDate = taskDate.AddDate(0, 1, 0)
			}
			//
		}

		found = false
		for {
			for _, v := range daysNum {
				if v < 0 {
					v = v + 1
					lastDay := time.Date(taskDate.Year(), taskDate.Month()+1, 0, 0, 0, 0, 0, time.UTC)
					v = lastDay.AddDate(0, 0, v).Day()
				}

				if taskDate.Day() == v {
					found = true
					break
				}
			}

			if found {
				break
			}

			taskDate = taskDate.AddDate(0, 0, 1)
		}
		return taskDate.Format(DateFormat), nil
	case t.Repeat == "":
		return "", nil
	default:
		return "", fmt.Errorf("неверный формат поля 't.Repeat': %s", t.Repeat)
	}
}
