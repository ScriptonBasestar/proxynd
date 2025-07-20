package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewCompletionCmd 자동완성 스크립트 생성 명령어
func NewCompletionCmd(rootCmd *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion",
		Short: "쉘 자동완성 스크립트 생성",
		Long: `지정된 쉘에 대한 자동완성 스크립트를 생성합니다.

이 스크립트를 사용하면 proxyndctl 명령어의 자동완성을 활성화할 수 있습니다.
bash, zsh, fish, powershell 쉘을 지원합니다.`,
	}

	// Bash 자동완성
	bashCmd := &cobra.Command{
		Use:   "bash",
		Short: "Bash 쉘 자동완성 스크립트 생성",
		Long: `Bash 쉘용 자동완성 스크립트를 생성합니다.

이 스크립트를 사용하려면:

1. 직접 실행하여 현재 쉘 세션에 로드:
   $ source <(proxyndctl completion bash)

2. 모든 새 세션에서 자동으로 로드하도록 설정:

   Linux:
   $ proxyndctl completion bash > /etc/bash_completion.d/proxyndctl

   macOS:
   $ proxyndctl completion bash > $(brew --prefix)/etc/bash_completion.d/proxyndctl

3. 사용자별 설정 (Linux/macOS):
   $ proxyndctl completion bash >> ~/.bashrc
   또는
   $ proxyndctl completion bash >> ~/.bash_profile

설정 후 새 쉘을 시작하거나 현재 쉘을 다시 로드하세요:
   $ source ~/.bashrc`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash"},
		Args:                  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return rootCmd.GenBashCompletion(os.Stdout)
		},
	}

	// Zsh 자동완성
	zshCmd := &cobra.Command{
		Use:   "zsh",
		Short: "Zsh 쉘 자동완성 스크립트 생성",
		Long: `Zsh 쉘용 자동완성 스크립트를 생성합니다.

이 스크립트를 사용하려면:

1. 직접 실행하여 현재 쉘 세션에 로드:
   $ source <(proxyndctl completion zsh)

2. 모든 새 세션에서 자동으로 로드하도록 설정:

   Zsh 자동완성 디렉토리에 저장:
   $ proxyndctl completion zsh > "${fpath[1]}/_proxyndctl"

   또는 ~/.zshrc에 추가:
   $ echo "autoload -U compinit; compinit" >> ~/.zshrc
   $ proxyndctl completion zsh >> ~/.zshrc

3. Oh My Zsh 사용자:
   $ proxyndctl completion zsh > ~/.oh-my-zsh/custom/plugins/proxyndctl/_proxyndctl

설정 후 새 쉘을 시작하거나 현재 쉘을 다시 로드하세요:
   $ source ~/.zshrc`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"zsh"},
		Args:                  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return rootCmd.GenZshCompletion(os.Stdout)
		},
	}

	// Fish 자동완성
	fishCmd := &cobra.Command{
		Use:   "fish",
		Short: "Fish 쉘 자동완성 스크립트 생성",
		Long: `Fish 쉘용 자동완성 스크립트를 생성합니다.

이 스크립트를 사용하려면:

1. Fish 자동완성 디렉토리에 저장:
   $ proxyndctl completion fish > ~/.config/fish/completions/proxyndctl.fish

2. 시스템 전체 설정 (관리자 권한 필요):
   $ proxyndctl completion fish > /usr/share/fish/completions/proxyndctl.fish

설정 후 새 쉘을 시작하면 자동완성이 활성화됩니다.`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"fish"},
		Args:                  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return rootCmd.GenFishCompletion(os.Stdout, true)
		},
	}

	// PowerShell 자동완성
	powershellCmd := &cobra.Command{
		Use:   "powershell",
		Short: "PowerShell 자동완성 스크립트 생성",
		Long: `PowerShell용 자동완성 스크립트를 생성합니다.

이 스크립트를 사용하려면:

1. 직접 실행하여 현재 세션에 로드:
   PS> proxyndctl completion powershell | Out-String | Invoke-Expression

2. 모든 새 세션에서 자동으로 로드하도록 설정:
   PS> proxyndctl completion powershell >> $PROFILE

PowerShell 프로필이 없는 경우 먼저 생성:
   PS> New-Item -Path $PROFILE -Type File -Force

설정 후 새 PowerShell을 시작하면 자동완성이 활성화됩니다.`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"powershell"},
		Args:                  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		},
	}

	cmd.AddCommand(bashCmd)
	cmd.AddCommand(zshCmd)
	cmd.AddCommand(fishCmd)
	cmd.AddCommand(powershellCmd)

	return cmd
}

// InstallCompletionGuide 자동완성 설치 가이드 출력
func InstallCompletionGuide() {
	fmt.Print(`
자동완성 스크립트 설치 가이드
============================

proxyndctl은 다양한 쉘에 대한 자동완성을 지원합니다.

지원되는 쉘:
- bash
- zsh  
- fish
- powershell

빠른 설치:

Bash:
  $ source <(proxyndctl completion bash)

Zsh:
  $ source <(proxyndctl completion zsh)

Fish:
  $ proxyndctl completion fish > ~/.config/fish/completions/proxyndctl.fish

PowerShell:
  PS> proxyndctl completion powershell | Out-String | Invoke-Expression

자세한 설치 방법은 각 명령어의 도움말을 참조하세요:
  $ proxyndctl completion <shell> --help
`)
}
