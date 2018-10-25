//
// Copyright 2018 Orkus, Inc
// All Rights Reserved.
//
// @author: Denys Nahurnyi, Orkus, Inc.
// @email:  denys.nahurnyi@Blackthorn-vision.com
// ---------------------------------------------------------------------------
package dataMgr

import "fmt"

type Service interface {
	CreateUser(login, password string) (bool, error)
}

type service struct {
	envType string
}

func (s *service) CreateUser(login, password string) (bool, error) {
	fmt.Println("Hello create user, password: ", password)
	return true, nil
}
