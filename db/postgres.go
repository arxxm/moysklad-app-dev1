package db

import (
	"fmt"
	"strings"

	"github.com/MaLowBar/moysklad-app-template/storage"
)

type MyStorage struct {
	storage.PostgreStorage
}

type Rule struct {
	AccountId string
	Id        string
	CounId    string
	CounName  string
	Operand1  string
	ProjectId string
	Project   string
	Operand2  string
	Comment   string
	Operand3  string
	Purpose   string
	EiId      string
	EiName    string
	Nomer     int
}

func NewStorage(postgreStorage *storage.PostgreStorage) *MyStorage {
	return &MyStorage{*postgreStorage}
}

func (s *MyStorage) GetRules(accountId string) ([]Rule, error) {

	rules := []Rule{}

	rows, err := s.DB.Query("SELECT * FROM rules WHERE accountid = $1", accountId)
	if err != nil {
		return rules, err
	}
	defer rows.Close()

	for rows.Next() {
		r := Rule{}
		err := rows.Scan(&r.AccountId, &r.Id, &r.CounId, &r.CounName, &r.Operand1, &r.ProjectId, &r.Project, &r.Operand2, &r.Comment, &r.Operand3,
			&r.Purpose, &r.EiId, &r.EiName)
		if err != nil {
			fmt.Println(err)
			continue
		}
		rules = append(rules, r)
	}

	return rules, nil
}

func (s *MyStorage) AddRule(rule map[string]string) error {

	_, err := s.DB.Exec(`INSERT INTO rules (accountId, counterparty_id, counterparty_name, operand_1, project_id,
		project_name, operand_2, comment, operand_3, purpose, expenseitem_id, expenseitem_name) VALUES ($1, $2, $3, $4,
		$5, $6, $7, $8, $9, $10, $11, $12)`,
		rule["accountId"], rule["counId"],
		rule["counName"], rule["operand1"], rule["projectId"],
		rule["project"], rule["operand2"], rule["comment"], rule["operand3"],
		rule["purpose"], rule["eiId"], rule["eiName"])
	if err != nil {
		fmt.Printf("________ Add rule error: %v\n", err)
		return err
	}
	return nil
}

func (s *MyStorage) DeleteRuleById(accountId, ruleId string) error {
	accId := strings.TrimSpace(accountId)
	id := strings.TrimSpace(ruleId)

	_, err := s.DB.Exec(`DELETE FROM rules WHERE accountid = $1 and nomer = $2`, accId, id)
	if err != nil {
		return err
	}
	return nil
}
