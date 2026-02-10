# 55h

**SSH 설정** 항목을 탐색하고 관리하기 위한 컴팩트한 TUI입니다. **Go**로 작성되었으며 **tview/tcell**을 사용합니다.

**이름의 의미는?** `55h`는 한눈에 보면 `ssh`처럼 보이며, 터미널 도구에 잘 어울리는 간결한 이름입니다.

55h는 SSH 설정(`Include` 파일 포함)을 읽어 호스트를 빠르게 찾고, 연결/테스트/삭제와 같은 일반적인 작업을 빠른 터미널 UI에서 제공합니다.

## 기능

- SSH 설정과 포함된 파일의 `Host` 항목 탐색
- 선택한 호스트의 상세 보기를 포함한 퍼지 검색
- 테마 선택 (사용자 설정에 저장)
- 앱 내 작업:
  - 연결
  - 핑 / 연결 테스트
  - 호스트 항목 삭제

## 스크린샷

<!-- Screenshot: Main host list -->
<!-- Screenshot: Host detail view -->
<!-- Screenshot: Theme selector -->

## 설치

### Homebrew (macOS)

```bash
brew install dev-minsoo/tap/55h
```

### 소스에서 빌드

Go **1.21+**가 필요합니다.

```bash
go build -o 55h .
./55h
```

### SSH 설정 경로 재정의

```bash
SSH_CONFIG=/path/to/config ./55h
```

## 사용법

TUI 실행:

```bash
55h
```

기본적으로 55h는 `~/.ssh/config`를 로드하며, 발견되는 모든 `Include` 지시자를 따라갑니다.

## 키 바인딩

| 키 | 동작 |
|-----|--------|
| ↑ / ↓ | 호스트 목록 이동 |
| `:` | 검색 포커스 |
| `Esc` | 검색 종료 / 모달 닫기 |
| `Enter` | 연결 (선택한 별칭으로 시스템 `ssh` 실행) |
| `p` | 연결 테스트 / 핑 |
| `d` | 선택한 호스트 삭제 |
| `t` | 테마 선택기 열기 / 테마 저장 |
| `q` | 종료 |
| `?` | 도움말 (키 바인딩 표시) |

핑 / 테스트 실행 명령:

```bash
ssh -o ConnectTimeout=5 -o BatchMode=yes -o StrictHostKeyChecking=accept-new <alias> exit 0
```

## 동작 참고 사항

- 호스트를 삭제하면 해당 항목을 제공한 소스 파일에서 **`Host` 블록 전체**가 제거됩니다.
  - 소스 경로가 확인된 경우에만 UI에서 삭제가 허용됩니다.
- 연결 시 `syscall.Exec`를 사용하여 현재 프로세스를 시스템 `ssh` 바이너리로 대체합니다.
  - 현재 터미널 세션이 SSH 세션이 됩니다.

## CLI: `add ssh`

명령줄에서 직접 새로운 SSH 호스트 항목을 추가합니다.

### 사용법 (정확한 형식)

```text
55h add ssh user@host [-p port] [-i identity] [-J jump] [-o Key=Value ...] [--name alias]
```

### 플래그

- `user@host` 또는 `host`
  - 새 항목의 대상
- `-p <port>`
  - 포트
- `-i <identity>`
  - `IdentityFile` 경로
- `-J <jump>`
  - `ProxyJump` 값
- `-o Key=Value`
  - 추가 SSH 설정
  - 지원 키 (대소문자 무관):
    - `forwardagent` (`yes` | `no`)
    - `identitiesonly` (`yes` | `no`)
    - `serveraliveinterval` (정수)
    - `serveralivecountmax` (정수)
  - 알 수 없는 `-o` 키는 무시됩니다
- `--name <alias>`
  - `Host` 별칭을 명시적으로 설정
  - 대화형(TTY) 실행에서 `--name`을 생략하면 제안된 별칭을 묻는 프롬프트가 표시됩니다
  - 표준 입력이 TTY가 **아닌** 경우 `--name`은 필수입니다

### 예제

대화형 (`--name` 생략 시 별칭을 묻습니다):

```bash
55h add ssh alice@example.com -p 2222 -i ~/.ssh/id_rsa
```

비대화형 / 스크립트 (`--name` 제공):

```bash
55h add ssh example.com --name myhost -p 2200 -J jump.example.org
```

### 참고

- `add` 명령은 구성된 SSH 설정 파일에 `Host` 블록을 추가합니다
  - 필요 시 상위 디렉터리가 생성됩니다
- 로드된 설정(포함된 파일 포함) 어디에서든 중복된 별칭이 발견되면 추가를 거부합니다
- CLI 사용 문자열과 동작은 의도적으로 최소화되어 있으며, 프로그램의 파싱 규칙과 일치합니다

## 기여하기

기여, 이슈, 제안 모두 환영합니다.

변경 사항을 제출하려는 경우:

1. 저장소를 포크합니다
2. 기능 브랜치를 생성합니다
3. 변경 사항을 만듭니다
4. 명확한 설명과 함께 풀 리퀘스트를 엽니다

가능한 경우 다음과 같은 컨벤셔널 스타일의 커밋 접두사를 사용해 주세요:

- `feat:` 새로운 기능
- `fix:` 버그 수정
- `docs:` 문서만 변경
- `style:` 코드 의미 변경 없는 형식 수정
- `refactor:` 버그 수정이나 기능 추가가 아닌 코드 변경
- `perf:` 성능 개선
- `test:` 테스트 추가 또는 수정
- `build:` 빌드 시스템 또는 외부 의존성 변경
- `ci:` CI 설정 변경
- `chore:` 유지보수 및 도구

## 영감

- **k9s** — 훌륭한 TUI가 복잡한 설정 작업을 즐겁게 만들 수 있음을 보여준 프로젝트.

## 라이선스

MIT
