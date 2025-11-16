package e2e

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"testing"
	"time"
)

var base_url = "http://test-app:8080"

func TestMain(m *testing.M) {
	waitForService()
	os.Exit(m.Run())
}

func waitForService() {
	healthURL := base_url + "/health"
	maxWait := 5 * time.Minute
	checkInterval := 2 * time.Second

	startTime := time.Now()
	for {
		if time.Since(startTime) > maxWait {
			log.Fatalf("Service did not become ready in %v", maxWait)
		}

		resp, err := http.Get(healthURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			log.Println("Service is ready!")
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}

		log.Printf("Waiting for service to be ready at %s...", healthURL)
		time.Sleep(checkInterval)
	}
}

type E2ETest struct {
	*testing.T
}

func (t *E2ETest) unique(name string) string {
	return name + "-" + time.Now().Format("20060102150405.999999")
}

func TestE2E(t *testing.T) {
	tt := &E2ETest{t}
	t.Run("TeamHappyPath", tt.TestTeamHappyPath)
	t.Run("CreatePR", tt.TestCreatePR)
	t.Run("MergePR", tt.TestMergePR)
	t.Run("ReassignPR", tt.TestReassignPR)
	t.Run("SetUserActiveStatus", tt.TestSetUserActiveStatus)
	t.Run("GetUserReviews", tt.TestGetUserReviews)
	t.Run("AddExistingTeam", tt.TestAddExistingTeam)
	t.Run("GetNonExistingTeam", tt.TestGetNonExistingTeam)
	t.Run("SetStatusOfNonExistingUser", tt.TestSetStatusOfNonExistingUser)
	t.Run("CreatePRWithNonExistingAuthor", tt.TestCreatePRWithNonExistingAuthor)
	t.Run("CreateExistingPR", tt.TestCreateExistingPR)
	t.Run("MergeNonExistingPR", tt.TestMergeNonExistingPR)
	t.Run("ReassignOnNonExistingPR", tt.TestReassignOnNonExistingPR)
	t.Run("ReassignNotAssignedReviewer", tt.TestReassignNotAssignedReviewer)
	t.Run("ReassignOnMergedPR", tt.TestReassignOnMergedPR)
}

func (t *E2ETest) TestTeamHappyPath(subT *testing.T) {
	// Team data
	teamName := t.unique("backend")
	team := map[string]interface{}{
		"team_name": teamName,
		"members": []map[string]interface{}{
			{"user_id": t.unique("u1"), "username": "Alice", "is_active": true},
			{"user_id": t.unique("u2"), "username": "Bob", "is_active": true},
		},
	}
	teamJSON, _ := json.Marshal(team)

	// 1. Add team
	resp, err := http.Post(base_url+"/team/add", "application/json", bytes.NewBuffer(teamJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 201, got %d. Body: %s", resp.StatusCode, body)
	}

	// 2. Get team
	resp, err = http.Get(base_url + "/team/get?team_name=" + teamName)
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 200, got %d. Body: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		subT.Fatalf("Failed to decode response: %v", err)
	}

	if result["team_name"] != teamName {
		subT.Errorf("Expected team_name '%s', got '%s'", teamName, result["team_name"])
	}
}

func (t *E2ETest) TestCreatePR(subT *testing.T) {
	// Team data
	teamName := t.unique("create-pr-team")
	authorID := t.unique("u3")
	team := map[string]interface{}{
		"team_name": teamName,
		"members": []map[string]interface{}{
			{"user_id": authorID, "username": "User3", "is_active": true},
		},
	}
	teamJSON, _ := json.Marshal(team)

	// Add team
	resp, err := http.Post(base_url+"/team/add", "application/json", bytes.NewBuffer(teamJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 201, got %d. Body: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// PR data
	prID := t.unique("pr")
	pr := map[string]interface{}{
		"pull_request_id":   prID,
		"pull_request_name": "Fix critical bug",
		"author_id":         authorID,
	}
	prJSON, _ := json.Marshal(pr)

	// Create PR
	resp, err = http.Post(base_url+"/pullRequest/create", "application/json", bytes.NewBuffer(prJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 201, got %d. Body: %s", resp.StatusCode, body)
	}

	var result map[string]map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		subT.Fatalf("Failed to decode response: %v", err)
	}

	if result["pr"]["pull_request_id"] != prID {
		subT.Errorf("Expected pr id '%s', got '%s'", prID, result["pr"]["pull_request_id"])
	}
}

func (t *E2ETest) TestMergePR(subT *testing.T) {
	// Team data
	teamName := t.unique("merge-pr-team")
	authorID := t.unique("u4")
	team := map[string]interface{}{
		"team_name": teamName,
		"members": []map[string]interface{}{
			{"user_id": authorID, "username": "User4", "is_active": true},
		},
	}
	teamJSON, _ := json.Marshal(team)

	// Add team
	resp, err := http.Post(base_url+"/team/add", "application/json", bytes.NewBuffer(teamJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 201, got %d. Body: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// PR data
	prID := t.unique("pr-merge")
	pr := map[string]interface{}{
		"pull_request_id":   prID,
		"pull_request_name": "Merge test",
		"author_id":         authorID,
	}
	prJSON, _ := json.Marshal(pr)

	// Create PR
	resp, err = http.Post(base_url+"/pullRequest/create", "application/json", bytes.NewBuffer(prJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 201, got %d. Body: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// Merge PR
	merge := map[string]interface{}{
		"pull_request_id": prID,
	}
	mergeJSON, _ := json.Marshal(merge)
	resp, err = http.Post(base_url+"/pullRequest/merge", "application/json", bytes.NewBuffer(mergeJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 200, got %d. Body: %s", resp.StatusCode, body)
	}

	var result map[string]map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		subT.Fatalf("Failed to decode response: %v", err)
	}

	if result["pr"]["status"] != "MERGED" {
		subT.Errorf("Expected pr status 'MERGED', got '%s'", result["pr"]["status"])
	}
}

func (t *E2ETest) TestReassignPR(subT *testing.T) {
	// Team data
	teamName := t.unique("reassign-pr-team")
	authorID := t.unique("u10")
	user11ID := t.unique("u11")
	user12ID := t.unique("u12")
	user13ID := t.unique("u13")
	team := map[string]interface{}{
		"team_name": teamName,
		"members": []map[string]interface{}{
			{"user_id": authorID, "username": "User10", "is_active": true},
			{"user_id": user11ID, "username": "User11", "is_active": true},
			{"user_id": user12ID, "username": "User12", "is_active": true},
			{"user_id": user13ID, "username": "User13", "is_active": true},
		},
	}
	teamJSON, _ := json.Marshal(team)

	// Add team
	resp, err := http.Post(base_url+"/team/add", "application/json", bytes.NewBuffer(teamJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 201, got %d. Body: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// PR data
	prID := t.unique("pr-reassign")
	pr := map[string]interface{}{
		"pull_request_id":   prID,
		"pull_request_name": "Reassign test",
		"author_id":         authorID,
	}
	prJSON, _ := json.Marshal(pr)

	// Create PR
	resp, err = http.Post(base_url+"/pullRequest/create", "application/json", bytes.NewBuffer(prJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 201, got %d. Body: %s", resp.StatusCode, body)
	}

	var createResp map[string]map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		subT.Fatalf("Failed to decode response: %v", err)
	}
	resp.Body.Close()

	reviewers := createResp["pr"]["assigned_reviewers"].([]interface{})
	if len(reviewers) == 0 {
		subT.Fatalf("No reviewers assigned, cannot test reassign")
	}
	oldReviewer := reviewers[0].(string)

	// Reassign PR
	reassign := map[string]interface{}{
		"pull_request_id": prID,
		"old_reviewer_id": oldReviewer,
	}
	reassignJSON, _ := json.Marshal(reassign)

	resp, err = http.Post(base_url+"/pullRequest/reassign", "application/json", bytes.NewBuffer(reassignJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 200, got %d. Body: %s", resp.StatusCode, body)
	}

	var reassignResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&reassignResp); err != nil {
		subT.Fatalf("Failed to decode response: %v", err)
	}

	newReviewer := reassignResp["replaced_by"].(string)
	if newReviewer == oldReviewer {
		subT.Errorf("Expected a new reviewer, but got the same one")
	}
}

func (t *E2ETest) TestSetUserActiveStatus(subT *testing.T) {
	// Team data
	teamName := t.unique("status-test-team")
	userID := t.unique("u20")
	team := map[string]interface{}{
		"team_name": teamName,
		"members": []map[string]interface{}{
			{"user_id": userID, "username": "User20", "is_active": true},
		},
	}
	teamJSON, _ := json.Marshal(team)

	// Add team
	resp, err := http.Post(base_url+"/team/add", "application/json", bytes.NewBuffer(teamJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 201, got %d. Body: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// Set user status
	status := map[string]interface{}{
		"user_id":   userID,
		"is_active": false,
	}
	statusJSON, _ := json.Marshal(status)

	resp, err = http.Post(base_url+"/users/setIsActive", "application/json", bytes.NewBuffer(statusJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 200, got %d. Body: %s", resp.StatusCode, body)
	}

	var result map[string]map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		subT.Fatalf("Failed to decode response: %v", err)
	}

	if result["user"]["is_active"] != false {
		subT.Errorf("Expected is_active to be false, got %v", result["user"]["is_active"])
	}
}

func (t *E2ETest) TestGetUserReviews(subT *testing.T) {
	// Team data
	teamName := t.unique("review-test-team")
	authorID := t.unique("u30")
	reviewerID := t.unique("u31")
	team := map[string]interface{}{
		"team_name": teamName,
		"members": []map[string]interface{}{
			{"user_id": authorID, "username": "User30", "is_active": true},
			{"user_id": reviewerID, "username": "User31", "is_active": true},
		},
	}
	teamJSON, _ := json.Marshal(team)

	// Add team
	resp, err := http.Post(base_url+"/team/add", "application/json", bytes.NewBuffer(teamJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 201, got %d. Body: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// PR data
	prID := t.unique("pr-review")
	pr := map[string]interface{}{
		"pull_request_id":   prID,
		"pull_request_name": "Review test",
		"author_id":         authorID,
	}
	prJSON, _ := json.Marshal(pr)

	// Create PR
	resp, err = http.Post(base_url+"/pullRequest/create", "application/json", bytes.NewBuffer(prJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 201, got %d. Body: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// Get user reviews
	resp, err = http.Get(base_url + "/users/getReview?user_id=" + reviewerID)
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 200, got %d. Body: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		subT.Fatalf("Failed to decode response: %v", err)
	}

	if result["user_id"] != reviewerID {
		subT.Errorf("Expected user_id '%s', got '%s'", reviewerID, result["user_id"])
	}

	prs := result["pull_requests"].([]interface{})
	if len(prs) != 1 {
		subT.Errorf("Expected 1 PR, got %d", len(prs))
	}

	if prs[0].(map[string]interface{})["pull_request_id"] != prID {
		subT.Errorf("Expected pr id '%s', got '%s'", prID, prs[0].(map[string]interface{})["pull_request_id"])
	}
}

func (t *E2ETest) TestAddExistingTeam(subT *testing.T) {
	// Team data
	teamName := t.unique("existing-team")
	team := map[string]interface{}{
		"team_name": teamName,
		"members": []map[string]interface{}{
			{"user_id": t.unique("u40"), "username": "User40", "is_active": true},
		},
	}
	teamJSON, _ := json.Marshal(team)

	// Add team
	resp, err := http.Post(base_url+"/team/add", "application/json", bytes.NewBuffer(teamJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 201, got %d. Body: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// Add the same team again
	resp, err = http.Post(base_url+"/team/add", "application/json", bytes.NewBuffer(teamJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 400, got %d. Body: %s", resp.StatusCode, body)
	}
}

func (t *E2ETest) TestGetNonExistingTeam(subT *testing.T) {
	resp, err := http.Get(base_url + "/team/get?team_name=" + t.unique("non-existing-team"))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 404, got %d. Body: %s", resp.StatusCode, body)
	}
}

func (t *E2ETest) TestSetStatusOfNonExistingUser(subT *testing.T) {
	status := map[string]interface{}{
		"user_id":   t.unique("non-existing-user"),
		"is_active": false,
	}
	statusJSON, _ := json.Marshal(status)

	resp, err := http.Post(base_url+"/users/setIsActive", "application/json", bytes.NewBuffer(statusJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 404, got %d. Body: %s", resp.StatusCode, body)
	}
}

func (t *E2ETest) TestCreatePRWithNonExistingAuthor(subT *testing.T) {
	pr := map[string]interface{}{
		"pull_request_id":   t.unique("pr-non-existing-author"),
		"pull_request_name": "Test PR",
		"author_id":         t.unique("non-existing-user"),
	}
	prJSON, _ := json.Marshal(pr)

	resp, err := http.Post(base_url+"/pullRequest/create", "application/json", bytes.NewBuffer(prJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 404, got %d. Body: %s", resp.StatusCode, body)
	}
}

func (t *E2ETest) TestCreateExistingPR(subT *testing.T) {
	// Team data
	teamName := t.unique("existing-pr-team")
	authorID := t.unique("u50")
	team := map[string]interface{}{
		"team_name": teamName,
		"members": []map[string]interface{}{
			{"user_id": authorID, "username": "User50", "is_active": true},
		},
	}
	teamJSON, _ := json.Marshal(team)

	// Add team
	resp, err := http.Post(base_url+"/team/add", "application/json", bytes.NewBuffer(teamJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 201, got %d. Body: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// PR data
	prID := t.unique("pr-existing")
	pr := map[string]interface{}{
		"pull_request_id":   prID,
		"pull_request_name": "Test PR",
		"author_id":         authorID,
	}
	prJSON, _ := json.Marshal(pr)

	// Create PR
	resp, err = http.Post(base_url+"/pullRequest/create", "application/json", bytes.NewBuffer(prJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 201, got %d. Body: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// Create the same PR again
	resp, err = http.Post(base_url+"/pullRequest/create", "application/json", bytes.NewBuffer(prJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 409, got %d. Body: %s", resp.StatusCode, body)
	}
}

func (t *E2ETest) TestMergeNonExistingPR(subT *testing.T) {
	pr := map[string]interface{}{
		"pull_request_id": t.unique("pr-non-existing"),
	}
	prJSON, _ := json.Marshal(pr)

	resp, err := http.Post(base_url+"/pullRequest/merge", "application/json", bytes.NewBuffer(prJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 404, got %d. Body: %s", resp.StatusCode, body)
	}
}

func (t *E2ETest) TestReassignOnNonExistingPR(subT *testing.T) {
	reassign := map[string]interface{}{
		"pull_request_id": t.unique("pr-non-existing"),
		"old_reviewer_id": t.unique("u1"),
	}
	reassignJSON, _ := json.Marshal(reassign)

	resp, err := http.Post(base_url+"/pullRequest/reassign", "application/json", bytes.NewBuffer(reassignJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 404, got %d. Body: %s", resp.StatusCode, body)
	}
}

func (t *E2ETest) TestReassignNotAssignedReviewer(subT *testing.T) {
	// Team data
	teamName := t.unique("reassign-not-assigned-team")
	authorID := t.unique("u60")
	user61ID := t.unique("u61")
	team := map[string]interface{}{
		"team_name": teamName,
		"members": []map[string]interface{}{
			{"user_id": authorID, "username": "User60", "is_active": true},
			{"user_id": user61ID, "username": "User61", "is_active": true},
		},
	}
	teamJSON, _ := json.Marshal(team)

	// Add team
	resp, err := http.Post(base_url+"/team/add", "application/json", bytes.NewBuffer(teamJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 201, got %d. Body: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// PR data
	prID := t.unique("pr-reassign-not-assigned")
	pr := map[string]interface{}{
		"pull_request_id":   prID,
		"pull_request_name": "Test PR",
		"author_id":         authorID,
	}
	prJSON, _ := json.Marshal(pr)

	// Create PR
	resp, err = http.Post(base_url+"/pullRequest/create", "application/json", bytes.NewBuffer(prJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 201, got %d. Body: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// Reassign a user who is not a reviewer
	reassign := map[string]interface{}{
		"pull_request_id": prID,
		"old_reviewer_id": user61ID,
	}
	reassignJSON, _ := json.Marshal(reassign)

	resp, err = http.Post(base_url+"/pullRequest/reassign", "application/json", bytes.NewBuffer(reassignJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 409, got %d. Body: %s", resp.StatusCode, body)
	}
}

func (t *E2ETest) TestReassignOnMergedPR(subT *testing.T) {
	// Team data
	teamName := t.unique("reassign-merged-pr-team")
	authorID := t.unique("u70")
	user71ID := t.unique("u71")
	team := map[string]interface{}{
		"team_name": teamName,
		"members": []map[string]interface{}{
			{"user_id": authorID, "username": "User70", "is_active": true},
			{"user_id": user71ID, "username": "User71", "is_active": true},
		},
	}
	teamJSON, _ := json.Marshal(team)

	// Add team
	resp, err := http.Post(base_url+"/team/add", "application/json", bytes.NewBuffer(teamJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 201, got %d. Body: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// PR data
	prID := t.unique("pr-reassign-merged")
	pr := map[string]interface{}{
		"pull_request_id":   prID,
		"pull_request_name": "Test PR",
		"author_id":         authorID,
	}
	prJSON, _ := json.Marshal(pr)

	// Create PR
	resp, err = http.Post(base_url+"/pullRequest/create", "application/json", bytes.NewBuffer(prJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 201, got %d. Body: %s", resp.StatusCode, body)
	}
	var createResp map[string]map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		subT.Fatalf("Failed to decode response: %v", err)
	}
	resp.Body.Close()

	reviewers := createResp["pr"]["assigned_reviewers"].([]interface{})
	if len(reviewers) == 0 {
		subT.Fatalf("No reviewers assigned, cannot test reassign")
	}
	oldReviewer := reviewers[0].(string)

	// Merge PR
	merge := map[string]interface{}{
		"pull_request_id": prID,
	}
	mergeJSON, _ := json.Marshal(merge)
	resp, err = http.Post(base_url+"/pullRequest/merge", "application/json", bytes.NewBuffer(mergeJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 200, got %d. Body: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// Reassign a reviewer on the merged PR
	reassign := map[string]interface{}{
		"pull_request_id": prID,
		"old_reviewer_id": oldReviewer,
	}
	reassignJSON, _ := json.Marshal(reassign)

	resp, err = http.Post(base_url+"/pullRequest/reassign", "application/json", bytes.NewBuffer(reassignJSON))
	if err != nil {
		subT.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		body, _ := ioutil.ReadAll(resp.Body)
		subT.Fatalf("Expected status 409, got %d. Body: %s", resp.StatusCode, body)
	}
}
