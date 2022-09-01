package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	moyskladapptemplate "github.com/MaLowBar/moysklad-app-template"
	"github.com/MaLowBar/moysklad-app-template/jsonapi"
	"github.com/MaLowBar/moysklad-app-template/vendorapi"
	"github.com/arxxm/moysklad-app-template-dev1/db"
	"github.com/labstack/echo/v4"
)

type exemplar struct {
	agentList   []jsonapi.Some
	eiList      []jsonapi.Some
	projectList []jsonapi.Some
	accountId   string
}

func newExemplar() *exemplar {
	return &exemplar{}
}

func iframeHandlerFunc(e *exemplar, myStorage *db.MyStorage, info moyskladapptemplate.AppConfig) moyskladapptemplate.AppHandler {

	iframeHandler := moyskladapptemplate.AppHandler{
		Method: "GET",
		Path:   "/go-apps/dev1/iframe",
		HandlerFunc: func(c echo.Context) error {

			operand1 := [3]string{"", "И", "ИЛИ"}
			operand := [3]string{"", "И СОДЕРЖИТ", "И НЕ СОДЕРЖИТ"}

			userContext, err := vendorapi.GetUserContext(c.QueryParam("contextKey"), info)
			if err != nil {
				return &echo.HTTPError{
					Code:    http.StatusInternalServerError,
					Message: err,
				}
			}

			e.accountId = userContext.AccountID

			rulesList, err := myStorage.GetRules(strings.TrimSpace(e.accountId))
			if err != nil {
				return err
			}
			n := 1
			for k := range rulesList {
				rulesList[k].Nomer = n
				n++
			}

			counterparties, err := jsonapi.GetAllEntities[jsonapi.Counterparty](myStorage, e.accountId, "counterparty", "")
			if err != nil {
				return &echo.HTTPError{
					Code:    http.StatusInternalServerError,
					Message: err,
				}
			}
			for _, c := range counterparties {
				e.agentList = append(e.agentList, c)
			}

			expenseItems, err := jsonapi.GetAllEntities[jsonapi.ExpenseItem](myStorage, e.accountId, "expenseitem", "")
			if err != nil {
				return &echo.HTTPError{
					Code:    http.StatusInternalServerError,
					Message: err,
				}
			}
			for _, ei := range expenseItems {
				e.eiList = append(e.eiList, ei)
			}

			projects, err := jsonapi.GetAllEntities[jsonapi.Project](myStorage, e.accountId, "project", "")
			if err != nil {
				return &echo.HTTPError{
					Code:    http.StatusInternalServerError,
					Message: err,
				}
			}
			for _, p := range projects {
				e.projectList = append(e.projectList, p)
			}

			return c.Render(http.StatusOK, "iframe.html", map[string]interface{}{
				"fullName":          userContext.FullName,
				"accountId":         e.accountId,
				"rulesList":         rulesList,
				"operand1":          operand1,
				"operand2":          operand,
				"operand3":          operand,
				"expenseItems":      expenseItems,
				"projects":          projects,
				"conterpartiesList": counterparties,
			})
		},
	}

	return iframeHandler
}

func addRuleHandlerFunc(e *exemplar, myStorage *db.MyStorage) moyskladapptemplate.AppHandler {

	addRuleHandler := moyskladapptemplate.AppHandler{
		Method: "POST",
		Path:   "/go-apps/dev1/post-rules",
		HandlerFunc: func(c echo.Context) error {

			counterparty := c.FormValue("counterparty")
			operand1Str := c.FormValue("operand1")
			operand2Str := c.FormValue("operand2")
			operand3Str := c.FormValue("operand3")
			project := c.FormValue("project")
			comment := c.FormValue("comment")
			purpose := c.FormValue("purpose")
			expenseitem := c.FormValue("expenseitem")

			counName, err := findEntitysName(e.agentList, counterparty)
			if err != nil {
				return err
			}
			projectName, err := findEntitysName(e.projectList, project)
			if err != nil {
				return err
			}
			eiName, err := findEntitysName(e.eiList, expenseitem)
			if err != nil {
				return err
			}

			var rule = make(map[string]string)
			rule["accountId"] = c.FormValue("accountId")
			rule["counId"] = counterparty
			rule["counName"] = counName
			rule["operand1"] = operand1Str
			rule["projectId"] = project
			rule["project"] = projectName
			rule["operand2"] = operand2Str
			rule["comment"] = comment
			rule["operand3"] = operand3Str
			rule["purpose"] = purpose
			rule["eiId"] = expenseitem
			rule["eiName"] = eiName

			err = myStorage.AddRule(rule)
			if err != nil {
				return &echo.HTTPError{
					Message: err.Error(),
				}
			}
			return c.Render(http.StatusOK, "added", nil)
		},
	}

	return addRuleHandler
}

func webhookHandlerFunc(e *exemplar, myStorage *db.MyStorage) moyskladapptemplate.AppHandler {

	webhooksHandler := moyskladapptemplate.AppHandler{
		Method: "POST",
		Path:   "/go-apps/dev1/webhook-processor",
		HandlerFunc: func(c echo.Context) error {

			webhookResponse := jsonapi.WebhookResponse{}
			err := json.NewDecoder(c.Request().Body).Decode(&webhookResponse)
			if err != nil {
				return err
			}

			if len(webhookResponse.Events) > 0 {
				for _, v := range webhookResponse.Events {
					accountId := v.AccountId
					rules, err := myStorage.GetRules(accountId)
					if err != nil {
						return err
					}
					accessToken, err := myStorage.AccessTokenByAccountId(accountId)
					if err != nil {
						return err
					}
					essence, err := getEssence(v.Meta.Href, accessToken)
					if err != nil {
						return err
					}
					err = compareAndChange(rules, accessToken, v.Meta.EssenceType, essence)
					if err != nil {
						return err
					}
				}
			}
			return c.Render(http.StatusOK, "", nil)
		},
	}

	return webhooksHandler
}

func runOnAllPaymentsHandlerFunc(e *exemplar, myStorage *db.MyStorage) moyskladapptemplate.AppHandler {

	runOnAllPayments := moyskladapptemplate.AppHandler{
		Method: "POST",
		Path:   "/go-apps/dev1/manual-start",
		HandlerFunc: func(c echo.Context) error {

			startDate := c.FormValue("startDate")
			endDate := c.FormValue("endDate")
			if len(startDate) == 0 || len(endDate) == 0 {
				return c.Render(http.StatusOK, "onfilter", map[string]interface{}{
					"data": nil})
			}

			rules, err := myStorage.GetRules(e.accountId)
			if err != nil {
				return err
			}
			accessToken, err := myStorage.AccessTokenByAccountId(e.accountId)
			if err != nil {
				return err
			}

			filter := fmt.Sprintf("moment>%s 00:00:00;moment<%s 23:59:59", startDate, endDate)

			cashouts, err := jsonapi.GetAllEntities[jsonapi.Entity](myStorage, e.accountId, "cashout", filter)
			if err != nil {
				return err
			}
			paymentouts, err := jsonapi.GetAllEntities[jsonapi.Entity](myStorage, e.accountId, "paymentout", filter)
			if err != nil {
				return err
			}

			for _, cashout := range cashouts {
				compareAndChange(rules, accessToken, "cashout", cashout)
			}
			for _, paymentout := range paymentouts {
				compareAndChange(rules, accessToken, "paymentout", paymentout)
			}
			data := [1]string{""}
			return c.Render(http.StatusOK, "onfilter", map[string]interface{}{
				"data": data})
		},
	}

	return runOnAllPayments
}

func delRuleHandlerFunc(e *exemplar, myStorage *db.MyStorage) moyskladapptemplate.AppHandler {

	delRuleHandler := moyskladapptemplate.AppHandler{
		Method: "POST",
		Path:   "/go-apps/dev1/delete-rule",
		HandlerFunc: func(c echo.Context) error {

			id := c.FormValue("rule_id")
			err := myStorage.DeleteRuleById(e.accountId, id)
			if err != nil {
				return err
			}
			return c.Render(http.StatusOK, "deleted", nil)
		},
	}

	return delRuleHandler
}
