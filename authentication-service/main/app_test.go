package main

import (
	"assignment1/internal/data"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func runTestServer() (*httptest.Server, func()) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=UTC", host, user, password, "testDB", port)
	db = initDB(dsn)
	db.AutoMigrate(&data.UserInfo{})

	r := setupRoutes(db)

	ts := httptest.NewServer(r)

	// Return the server and a cleanup function
	return ts, func() {
		ts.Close()
		db.Migrator().DropTable(&data.UserInfo{})
	}
}

func TestRegister(t *testing.T) {
	ts, cleanup := runTestServer()
	defer ts.Close()

	body := map[string]string{
		"email":         "karataev020902@gmail.com",
		"fname":         "dkcreator",
		"password_hash": "password",
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Could not encode body: %v", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/register", ts.URL), bytes.NewBuffer(bodyBytes))
	if err != nil {
		t.Fatalf("Could not create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Could not send request: %v", err)
	}
	defer resp.Body.Close()

	//if resp.StatusCode != http.StatusCreated {
	//	t.Fatalf("Expected status Created; got %v", resp.StatusCode)
	//}

	var user data.UserInfo
	if err := db.Where("email = ?", "karataev020902@gmail.com").First(&user).Error; err != nil {
		t.Fatalf("Could not find user in database: %v", err)
	}
	cleanup()
}

func TestLogin(t *testing.T) {
	ts, cleanup := runTestServer()
	defer cleanup()

	registerBody := map[string]string{
		"email":         "testuser@example.com",
		"fname":         "Test User",
		"password_hash": "password",
	}
	registerBodyBytes, err := json.Marshal(registerBody)
	if err != nil {
		t.Fatalf("Could not encode body: %v", err)
	}

	_, err = http.Post(ts.URL+"/register", "application/json", bytes.NewBuffer(registerBodyBytes))
	if err != nil {
		t.Fatalf("Could not register user: %v", err)
	}

	loginBody := map[string]string{
		"email":         "testuser@example.com",
		"password_hash": "password",
	}
	loginBodyBytes, err := json.Marshal(loginBody)
	if err != nil {
		t.Fatalf("Could not encode body: %v", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/login", ts.URL), bytes.NewBuffer(loginBodyBytes))
	if err != nil {
		t.Fatalf("Could not create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Could not send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status OK; got %v", resp.StatusCode)
	}
}

func TestActivateHandler(t *testing.T) {
	ts, cleanup := runTestServer()
	defer cleanup()

	registerBody := map[string]string{
		"email":         "testuser@example.com",
		"fname":         "Test User",
		"password_hash": "password",
	}
	registerBodyBytes, err := json.Marshal(registerBody)
	if err != nil {
		t.Fatalf("Could not encode body: %v", err)
	}

	_, err = http.Post(ts.URL+"/register", "application/json", bytes.NewBuffer(registerBodyBytes))
	if err != nil {
		t.Fatalf("Could not register user: %v", err)
	}

	// Get the user from the database and check that they are not activated
	var user data.UserInfo
	if err := db.Where("email = ?", "testuser@example.com").First(&user).Error; err != nil {
		t.Fatalf("Could not find user in database: %v", err)
	}
	if user.Activated {
		t.Fatalf("Expected user to not be activated")

	}

	// Get the activation link from the database
	var activationLink string
	if err := db.Model(&data.UserInfo{}).Select("activation_link").Where("email = ?", "testuser@example.com").Scan(&activationLink).Error; err != nil {
		t.Fatalf("Could not find activation link in database: %v", err)
	}

	// Send a request to the activation link
	resp, err := http.Get(ts.URL + "/activate/" + activationLink)
	if err != nil {
		t.Fatalf("Could not send request: %v", err)
	}
	defer resp.Body.Close()

	// Get the user from the database and check that they are activated
	if err := db.Where("email = ?", "testuser@example.com").First(&user).Error; err != nil {
		t.Fatalf("Could not find user in database: %v", err)
	}
	if !user.Activated {
		t.Fatalf("Expected user to be activated")
	}
}

func TestGetAllUserInfoHandler(t *testing.T) {
	ts, cleanup := runTestServer()
	defer cleanup()

	registerBody := map[string]string{
		"email":         "admin@example.com",
		"fname":         "Admin User",
		"password_hash": "password",
	}
	registerBodyBytes, err := json.Marshal(registerBody)
	if err != nil {
		t.Fatalf("Could not encode body: %v", err)
	}

	_, err = http.Post(ts.URL+"/register", "application/json", bytes.NewBuffer(registerBodyBytes))
	if err != nil {
		t.Fatalf("Could not register user: %v", err)
	}

	// Activate the user and set role to ADMIN
	var user data.UserInfo
	if err := db.Where("email = ?", "admin@example.com").First(&user).Error; err != nil {
		t.Fatalf("Could not find user in database: %v", err)
	}
	user.Activated = true
	user.UserRole = "ADMIN"
	if err := db.Save(&user).Error; err != nil {
		t.Fatalf("Could not update user: %v", err)
	}

	loginBody := map[string]string{
		"email":         "admin@example.com",
		"password_hash": "password",
	}
	loginBodyBytes, err := json.Marshal(loginBody)
	if err != nil {
		t.Fatalf("Could not encode body: %v", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/login", ts.URL), bytes.NewBuffer(loginBodyBytes))
	if err != nil {
		t.Fatalf("Could not create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Could not send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status OK; got %v", resp.StatusCode)
	}

	// get token from login response, and add it to the header of the request to get all users
	var loginResponse map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&loginResponse); err != nil {
		t.Fatalf("Could not decode response: %v", err)
	}
	token := loginResponse["token"]

	req, err = http.NewRequest("GET", fmt.Sprintf("%s/api/users", ts.URL), nil)
	if err != nil {
		t.Fatalf("Could not create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Could not send request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status OK; got %v", resp.StatusCode)
	}

	cleanup()
}

// Admin role required route tests
func TestEditUserInfoHandler(t *testing.T) {
	ts, cleanup := runTestServer()
	defer cleanup()

	// Register and login as an admin user
	registerBody := map[string]string{
		"email":         "admin@example.com",
		"fname":         "Admin User",
		"password_hash": "password",
	}
	registerBodyBytes, err := json.Marshal(registerBody)
	if err != nil {
		t.Fatalf("Could not encode body: %v", err)
	}

	_, err = http.Post(ts.URL+"/register", "application/json", bytes.NewBuffer(registerBodyBytes))
	if err != nil {
		t.Fatalf("Could not register user: %v", err)
	}

	var user data.UserInfo
	if err := db.Where("email = ?", "admin@example.com").First(&user).Error; err != nil {
		t.Fatalf("Could not find user in database: %v", err)
	}
	user.Activated = true
	user.UserRole = "ADMIN"
	if err := db.Save(&user).Error; err != nil {
		t.Fatalf("Could not update user: %v", err)
	}

	loginBody := map[string]string{
		"email":         "admin@example.com",
		"password_hash": "password",
	}
	loginBodyBytes, err := json.Marshal(loginBody)
	if err != nil {
		t.Fatalf("Could not encode body: %v", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/login", ts.URL), bytes.NewBuffer(loginBodyBytes))
	if err != nil {
		t.Fatalf("Could not create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Could not send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status OK; got %v", resp.StatusCode)
	}

	var loginResponse map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&loginResponse); err != nil {
		t.Fatalf("Could not decode response: %v", err)
	}
	token := loginResponse["token"]

	// Edit a user
	editUserBody := map[string]string{
		"f_name": "Updated User",
		"s_name": "Updated Surname",
		"email":  "updated@example.com",
	}
	editUserBodyBytes, err := json.Marshal(editUserBody)
	if err != nil {
		t.Fatalf("Could not encode body: %v", err)
	}

	req, err = http.NewRequest("PUT", fmt.Sprintf("%s/api/admin/users/%d", ts.URL, user.ID), bytes.NewBuffer(editUserBodyBytes))
	if err != nil {
		t.Fatalf("Could not create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Could not send request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status OK; got %v", resp.StatusCode)
	}
}

func TestDeleteUserInfoHandler(t *testing.T) {
	ts, cleanup := runTestServer()
	defer cleanup()

	// Register and login as an admin user
	registerBody := map[string]string{
		"email":         "admin@example.com",
		"fname":         "Admin User",
		"password_hash": "password",
	}
	registerBodyBytes, err := json.Marshal(registerBody)
	if err != nil {
		t.Fatalf("Could not encode body: %v", err)
	}

	_, err = http.Post(ts.URL+"/register", "application/json", bytes.NewBuffer(registerBodyBytes))
	if err != nil {
		t.Fatalf("Could not register user: %v", err)
	}

	var user data.UserInfo
	if err := db.Where("email = ?", "admin@example.com").First(&user).Error; err != nil {
		t.Fatalf("Could not find user in database: %v", err)
	}
	user.Activated = true
	user.UserRole = "ADMIN"
	if err := db.Save(&user).Error; err != nil {
		t.Fatalf("Could not update user: %v", err)
	}

	loginBody := map[string]string{
		"email":         "admin@example.com",
		"password_hash": "password",
	}
	loginBodyBytes, err := json.Marshal(loginBody)
	if err != nil {
		t.Fatalf("Could not encode body: %v", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/login", ts.URL), bytes.NewBuffer(loginBodyBytes))
	if err != nil {
		t.Fatalf("Could not create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Could not send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status OK; got %v", resp.StatusCode)
	}

	var loginResponse map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&loginResponse); err != nil {
		t.Fatalf("Could not decode response: %v", err)
	}
	token := loginResponse["token"]

	// Delete a user
	req, err = http.NewRequest("DELETE", fmt.Sprintf("%s/api/admin/users/%d", ts.URL, user.ID), nil)
	if err != nil {
		t.Fatalf("Could not create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Could not send request: %v", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("Expected status No Content; got %v", resp.StatusCode)
	}
}
