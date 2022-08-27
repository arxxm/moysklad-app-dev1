package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/MaLowBar/moysklad-app-template/jsonapi"
	"github.com/MaLowBar/moysklad-app-template/utils"
	"github.com/arxxm/moysklad-app-template-dev1/db"
)

func findEntitysName(s []jsonapi.Some, id string) (string, error) {
	if id == "0" {
		return "", nil
	}
	formId := strings.TrimSpace(id)
	for _, n := range s {
		sliceId := strings.TrimSpace(n.GetID())
		if sliceId == formId {
			return n.GetName(), nil
		}
	}

	return "", errors.New("entity not found")
}

func compareAndChange(r []db.Rule, accessToken, eType string, essence jsonapi.Entity) error {

	strUrl := "https://online.moysklad.ru/api/remap/1.2/entity/" + eType

	agentHref := essence.Agent.Meta.Href
	agentId := ""
	if len(agentHref) != 0 {
		agentId = agentHref[len(agentHref)-36:]
	}

	projectHref := essence.Project.Meta.Href
	projectId := ""
	if len(projectHref) != 0 {
		projectId = projectHref[len(projectHref)-36:]
	}

	for _, rule := range r {
		mapRule := make(map[string]bool)
		if len(rule.CounId) != 0 {
			mapRule["agent"] = true
		}
		if len(rule.Operand1) != 0 {
			mapRule["operand1"] = true
		}
		if len(rule.Operand2) != 0 {
			mapRule["operand2"] = true
		}
		if len(rule.Operand3) != 0 {
			mapRule["operand3"] = true
		}

		if mapRule["agent"] && mapRule["operand1"] {
			if rule.Operand1 == "И" {
				if strings.TrimSpace(rule.CounId) == strings.TrimSpace(agentId) && strings.TrimSpace(rule.ProjectId) == strings.TrimSpace(projectId) {
				} else {
					mapRule["agent"] = false
					mapRule["operand1"] = false
				}
			} else if rule.Operand1 == "ИЛИ" {
				if strings.TrimSpace(rule.CounId) == strings.TrimSpace(agentId) || strings.TrimSpace(rule.ProjectId) == strings.TrimSpace(projectId) {

				} else {
					mapRule["agent"] = false
					mapRule["operand1"] = false
				}
			}
		} else if mapRule["agent"] && !mapRule["operand1"] {
			if strings.TrimSpace(rule.CounId) != strings.TrimSpace(agentId) {
				mapRule["agent"] = false
			}
		} else if !mapRule["agent"] && mapRule["operand1"] {
			if strings.TrimSpace(rule.ProjectId) != strings.TrimSpace(projectId) {
				mapRule["operand1"] = false
			}
		}

		if mapRule["operand2"] {
			if rule.Operand2 == "И СОДЕРЖИТ" {
				if !strings.Contains(strings.ToLower(essence.Description), strings.ToLower(rule.Comment)) {
					mapRule["operand2"] = false
				}
			} else if rule.Operand2 == "И НЕ СОДЕРЖИТ" {
				if strings.Contains(strings.ToLower(essence.Description), strings.ToLower(rule.Comment)) {
					mapRule["operand2"] = false
				}
			}
		}

		if mapRule["operand3"] {
			if rule.Operand3 == "И СОДЕРЖИТ" {
				if !strings.Contains(strings.ToLower(essence.PaymentPurpose), strings.ToLower(rule.Purpose)) {
					mapRule["operand3"] = false
				}
			} else if rule.Operand3 == "И НЕ СОДЕРЖИТ" {
				if strings.Contains(strings.ToLower(essence.PaymentPurpose), strings.ToLower(rule.Purpose)) {
					mapRule["operand3"] = false
				}
			}
		}

		change := true
		for _, v := range mapRule {
			if !v {
				change = false
			}
		}

		if change {
			id := strings.TrimSpace(essence.Id)
			err := changeEssence(strUrl+"/"+id, accessToken, rule.EiId)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func getEssence(url, accessToken string) (jsonapi.Entity, error) {

	essence := jsonapi.Entity{}
	req, err := utils.Request("GET", url, accessToken, nil)
	if err != nil {
		return essence, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return essence, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return essence, err
	}

	json.Unmarshal(body, &essence)

	return essence, nil
}

func changeEssence(url, accessToken, expenseItemID string) error {

	ei := jsonapi.ExpenseItemForReq{}
	ei.ExpenseItem.Meta.Href = "https://online.moysklad.ru/api/remap/1.2/entity/expenseitem/" + expenseItemID
	ei.ExpenseItem.Meta.MetaDataHref = "https://online.moysklad.ru/api/remap/1.2/entity/expenseitem/metadata"
	ei.ExpenseItem.Meta.Type = "expenseitem"
	ei.ExpenseItem.Meta.MediaType = "application/json"

	jsonBody, err := json.Marshal(ei)
	if err != nil {
		return err
	}

	req, err := utils.Request("PUT", url, accessToken, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	status := resp.StatusCode
	if status != 200 {
		return errors.New("statuscode is not 200")
	}

	return nil
}
