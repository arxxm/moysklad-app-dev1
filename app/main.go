package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	moyskladapptemplate "github.com/MaLowBar/moysklad-app-template"
	"github.com/MaLowBar/moysklad-app-template/storage"
	"github.com/arxxm/moysklad-app-template-dev1/db"
)

const (
	// Initialize connection constants.
	HOST     = "localhost"
	DATABASE = "dbexpenseitems"
	USER     = "dev1"
	PASSWORD = "111"
)

func main() {

	var info = moyskladapptemplate.AppConfig{
		ID:           "3f586619-ec7e-4464-b284-1169d9fa6958",
		UID:          "dev1.sorochinsky",
		SecretKey:    "ALN38ELqh6Iroor561bXKfSa1BwFlfGJNFlQFWrDQjDKTX0LdMWcxRcylAo3nmPVnpMqt1SIEtN1bYtoTk7EGMhJ5rYPi7mWNDjeS6yK9IlHSN5LLrMS3zE1p69TD9p3",
		VendorAPIURL: "/go-apps/dev1/api/moysklad/vendor/1.0/apps/:appId/:accountId",
	}

	var connectionString string = fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=require", HOST, USER, PASSWORD, DATABASE)
	// myStorage, err := storage.NewPostgreStorage(connectionString)
	postgreStorage, err := storage.NewPostgreStorage(connectionString)
	myStorage := db.NewStorage(postgreStorage)
	if err != nil {
		log.Fatal(fmt.Errorf("cannot create storage: %w", err))
		return
	}

	ex := newExemplar()

	// Определяем обработчики
	var iframeHandler = iframeHandlerFunc(ex, myStorage, info)

	var addRuleHandler = addRuleHandlerFunc(ex, myStorage)

	var webhooksHandler = webhookHandlerFunc(ex, myStorage)

	var delRuleHandler = delRuleHandlerFunc(ex, myStorage)

	var runOnAllPayments = runOnAllPaymentsHandlerFunc(ex, myStorage)

	//Формируем слайс с именами шаблонов. Например: []string{"header.html", "footer.html"}
	templateNames := []string{"deleted.html", "header.html", "footer.html", "onfilter.html", "added.html"}

	// Создаем приложение
	app := moyskladapptemplate.NewApp(&info, myStorage, templateNames, iframeHandler, addRuleHandler, webhooksHandler, delRuleHandler, runOnAllPayments)

	e := make(chan error)
	go func() {
		e <- app.Run("0.0.0.0:8003") // Запускаем
	}()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)

	select {
	case err = <-e:
		log.Printf("Server returned error: %s", err)
	case <-c:
		app.Stop(5)
		log.Println("Stop signal received")
	}
}
