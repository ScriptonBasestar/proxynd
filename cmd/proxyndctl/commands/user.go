package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// UserInfo CLI용 사용자 정보 구조체
type UserInfo struct {
	Username    string     `json:"username"`
	Role        string     `json:"role"`
	CreatedAt   time.Time  `json:"created_at"`
	LastLogin   *time.Time `json:"last_login,omitempty"`
	Description string     `json:"description,omitempty"`
	Active      bool       `json:"active"`
}

// UserListResponse 사용자 목록 응답
type UserListResponse struct {
	Users     []UserInfo `json:"users"`
	Total     int        `json:"total"`
	Timestamp time.Time  `json:"timestamp"`
}

// UserAddRequest 사용자 추가 요청
type UserAddRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	Role        string `json:"role,omitempty"`
	Description string `json:"description,omitempty"`
}

// UserDeleteRequest 사용자 삭제 요청
type UserDeleteRequest struct {
	Username string `json:"username"`
}

// NewUserCmd 사용자 관리 명령어 생성
func NewUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "사용자 관리 명령어",
		Long:  "ProxyND 서버의 사용자를 관리합니다 (추가, 삭제, 목록 조회).",
	}

	cmd.AddCommand(newUserListCmd())
	cmd.AddCommand(newUserAddCmd())
	cmd.AddCommand(newUserDeleteCmd())
	cmd.AddCommand(newUserInfoCmd())

	return cmd
}

// newUserListCmd 사용자 목록 조회 명령어
func newUserListCmd() *cobra.Command {
	var showDetails bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "사용자 목록 조회",
		Long:  "등록된 모든 사용자의 목록을 조회합니다.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runUserList(showDetails)
		},
	}

	cmd.Flags().BoolVarP(&showDetails, "details", "d", false, "상세 정보 표시")

	return cmd
}

// newUserAddCmd 사용자 추가 명령어
func newUserAddCmd() *cobra.Command {
	var (
		username    string
		password    string
		role        string
		description string
		interactive bool
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "사용자 추가",
		Long:  "새로운 사용자를 추가합니다.",
		RunE: func(_ *cobra.Command, _ []string) error {
			if interactive {
				return runUserAddInteractive()
			}
			return runUserAdd(username, password, role, description)
		},
	}

	cmd.Flags().StringVarP(&username, "username", "u", "", "사용자명")
	cmd.Flags().StringVarP(&password, "password", "p", "", "비밀번호")
	cmd.Flags().StringVarP(&role, "role", "r", "user", "역할 (admin, user)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "사용자 설명")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "대화형 모드")

	return cmd
}

// newUserDeleteCmd 사용자 삭제 명령어
func newUserDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <username>",
		Short: "사용자 삭제",
		Long:  "지정된 사용자를 삭제합니다.",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runUserDelete(args[0], force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "확인 없이 삭제")

	return cmd
}

// newUserInfoCmd 사용자 정보 조회 명령어
func newUserInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info <username>",
		Short: "사용자 정보 조회",
		Long:  "지정된 사용자의 상세 정보를 조회합니다.",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runUserInfo(args[0])
		},
	}

	return cmd
}

// runUserList 사용자 목록 조회 실행
func runUserList(showDetails bool) error {
	serverURL := getServerURL()
	apiURL := fmt.Sprintf("%s/api/user/list", serverURL)

	// API 호출
	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf(errServerConnection, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(errAPIRequest, resp.StatusCode)
	}

	// 응답 파싱
	var result UserListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf(errResponseParsing, err)
	}

	// 결과 출력
	outputFormat := getOutputFormat()
	switch outputFormat {
	case formatJSON:
		return outputJSON(result)
	case formatYAML:
		return outputYAML(result)
	default:
		return outputUserListTable(result, showDetails)
	}
}

// runUserAdd 사용자 추가 실행
func runUserAdd(username, password, role, description string) error {
	// 입력 검증
	if username == "" {
		return fmt.Errorf("사용자명이 필요합니다")
	}
	if password == "" {
		return fmt.Errorf("비밀번호가 필요합니다")
	}

	req := UserAddRequest{
		Username:    username,
		Password:    password,
		Role:        role,
		Description: description,
	}

	// JSON 인코딩
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("요청 데이터 인코딩 실패: %v", err)
	}

	// API 호출
	serverURL := getServerURL()
	apiURL := fmt.Sprintf("%s/api/user/add", serverURL)

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf(errServerConnection, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusCreated {
		fmt.Printf("✅ 사용자 '%s'가 성공적으로 추가되었습니다.\n", username)
		return nil
	}

	// 에러 응답 처리
	var errorMsg string
	if err := json.NewDecoder(resp.Body).Decode(&errorMsg); err == nil {
		return fmt.Errorf("사용자 추가 실패: %s", errorMsg)
	}
	return fmt.Errorf("사용자 추가 실패: HTTP %d", resp.StatusCode)
}

// runUserAddInteractive 대화형 사용자 추가
func runUserAddInteractive() error {
	fmt.Println("🧑‍💻 새 사용자 추가")
	fmt.Println()

	// 사용자명 입력
	var username string
	fmt.Print("사용자명: ")
	if _, err := fmt.Scanln(&username); err != nil {
		return fmt.Errorf("사용자명 입력 실패: %v", err)
	}

	// 비밀번호 입력 (숨김)
	fmt.Print("비밀번호: ")
	passwordBytes, err := term.ReadPassword(syscall.Stdin)
	if err != nil {
		return fmt.Errorf("비밀번호 입력 실패: %v", err)
	}
	password := string(passwordBytes)
	fmt.Println()

	// 비밀번호 확인
	fmt.Print("비밀번호 확인: ")
	confirmBytes, err := term.ReadPassword(syscall.Stdin)
	if err != nil {
		return fmt.Errorf("비밀번호 확인 입력 실패: %v", err)
	}
	confirm := string(confirmBytes)
	fmt.Println()

	if password != confirm {
		return fmt.Errorf("비밀번호가 일치하지 않습니다")
	}

	// 역할 입력
	var role string
	fmt.Print("역할 [user]: ")
	_, _ = fmt.Scanln(&role)
	if role == "" {
		role = "user"
	}

	// 설명 입력
	var description string
	fmt.Print("설명 (선택사항): ")
	_, _ = fmt.Scanln(&description)

	fmt.Println()

	return runUserAdd(username, password, role, description)
}

// runUserDelete 사용자 삭제 실행
func runUserDelete(username string, force bool) error {
	// 확인 (force 옵션이 없는 경우)
	if !force {
		fmt.Printf("사용자 '%s'를 정말 삭제하시겠습니까? (y/N): ", username)
		var confirm string
		_, _ = fmt.Scanln(&confirm)
		if confirm != "y" && confirm != "Y" {
			fmt.Println("삭제가 취소되었습니다.")
			return nil
		}
	}

	req := UserDeleteRequest{
		Username: username,
	}

	// JSON 인코딩
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("요청 데이터 인코딩 실패: %v", err)
	}

	// API 호출
	serverURL := getServerURL()
	apiURL := fmt.Sprintf("%s/api/user/delete", serverURL)

	client := &http.Client{}
	req2, err := http.NewRequest("DELETE", apiURL, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("요청 생성 실패: %v", err)
	}
	req2.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req2)
	if err != nil {
		return fmt.Errorf(errServerConnection, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("✅ 사용자 '%s'가 성공적으로 삭제되었습니다.\n", username)
		return nil
	}

	// 에러 응답 처리
	var errorMsg string
	if err := json.NewDecoder(resp.Body).Decode(&errorMsg); err == nil {
		return fmt.Errorf("사용자 삭제 실패: %s", errorMsg)
	}
	return fmt.Errorf("사용자 삭제 실패: HTTP %d", resp.StatusCode)
}

// runUserInfo 사용자 정보 조회 실행
func runUserInfo(username string) error {
	serverURL := getServerURL()
	apiURL := fmt.Sprintf("%s/api/user/%s", serverURL, username)

	// API 호출
	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf(errServerConnection, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("사용자 '%s'를 찾을 수 없습니다", username)
		}
		return fmt.Errorf(errAPIRequest, resp.StatusCode)
	}

	// 응답 파싱
	var result UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf(errResponseParsing, err)
	}

	// 결과 출력
	outputFormat := getOutputFormat()
	switch outputFormat {
	case formatJSON:
		return outputJSON(result)
	case formatYAML:
		return outputYAML(result)
	default:
		return outputUserInfoTable(result)
	}
}

// outputUserListTable 사용자 목록을 테이블 형태로 출력
func outputUserListTable(result UserListResponse, showDetails bool) error {
	fmt.Printf("👥 사용자 목록 (총 %d명)\n", result.Total)
	fmt.Printf("조회 시간: %s\n\n", result.Timestamp.Format(dateTimeFormat))

	if len(result.Users) == 0 {
		fmt.Println("등록된 사용자가 없습니다.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	if showDetails {
		_, _ = fmt.Fprintln(w, "사용자명\t역할\t상태\t생성일\t마지막로그인\t설명")
		_, _ = fmt.Fprintln(w, "--------\t----\t----\t------\t----------\t----")

		for _, user := range result.Users {
			status := iconSuccess
			if !user.Active {
				status = iconError
			}

			lastLogin := "-"
			if user.LastLogin != nil {
				lastLogin = user.LastLogin.Format(dateFormat)
			}

			description := user.Description
			if description == "" {
				description = "-"
			}

			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				user.Username,
				user.Role,
				status,
				user.CreatedAt.Format(dateFormat),
				lastLogin,
				description,
			)
		}
	} else {
		_, _ = fmt.Fprintln(w, "사용자명\t역할\t상태\t생성일")
		_, _ = fmt.Fprintln(w, "--------\t----\t----\t------")

		for _, user := range result.Users {
			status := iconSuccess
			if !user.Active {
				status = iconError
			}

			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				user.Username,
				user.Role,
				status,
				user.CreatedAt.Format(dateFormat),
			)
		}
	}

	_ = w.Flush()
	return nil
}

// outputUserInfoTable 사용자 정보를 테이블 형태로 출력
func outputUserInfoTable(user UserInfo) error {
	status := statusActive
	if !user.Active {
		status = statusInactive
	}

	lastLogin := "없음"
	if user.LastLogin != nil {
		lastLogin = user.LastLogin.Format(dateTimeFormat)
	}

	description := user.Description
	if description == "" {
		description = "없음"
	}

	fmt.Printf("👤 사용자 정보: %s\n\n", user.Username)
	fmt.Printf("역할: %s\n", user.Role)
	fmt.Printf("상태: %s\n", status)
	fmt.Printf("생성일: %s\n", user.CreatedAt.Format(dateTimeFormat))
	fmt.Printf("마지막 로그인: %s\n", lastLogin)
	fmt.Printf("설명: %s\n", description)

	return nil
}
