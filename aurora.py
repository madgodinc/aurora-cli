#!/usr/bin/env python3
"""Aurora AI — CLI-клиент для общения с Aurora."""

import sys
import os

# Fix Windows encoding
if sys.platform == "win32":
    sys.stdout.reconfigure(encoding="utf-8", errors="replace")
    sys.stderr.reconfigure(encoding="utf-8", errors="replace")
    os.environ.setdefault("PYTHONIOENCODING", "utf-8")
import json
import signal
import shutil

try:
    import httpx
except ImportError:
    print("Ошибка: установи httpx — pip install httpx")
    sys.exit(1)

__version__ = "0.4.0"

# ─── Цвета ────────────────────────────────────────────────────────────────────

PURPLE = "\033[35m"
GREEN = "\033[32m"
DIM = "\033[2m"
BOLD = "\033[1m"
RESET = "\033[0m"
CYAN = "\033[36m"
RED = "\033[31m"
YELLOW = "\033[33m"


# ─── Конфиг ───────────────────────────────────────────────────────────────────

CONFIG_DIR = os.path.expanduser("~/.aurora")
CONFIG_FILE = os.path.join(CONFIG_DIR, "config.json")
DEFAULT_SERVER = "https://fraylon.net"

# Режимы подтверждения файловых операций
APPROVE_ASK = "ask"        # Спрашивать каждый раз (по умолчанию)
APPROVE_AUTO = "auto"      # Не спрашивать, применять всё
_approve_mode = APPROVE_ASK


def load_config():
    if os.path.exists(CONFIG_FILE):
        try:
            with open(CONFIG_FILE, "r", encoding="utf-8") as f:
                return json.load(f)
        except Exception:
            pass
    return {"server": DEFAULT_SERVER}


def save_config(cfg):
    os.makedirs(CONFIG_DIR, exist_ok=True)
    with open(CONFIG_FILE, "w", encoding="utf-8") as f:
        json.dump(cfg, f, indent=2, ensure_ascii=False)


# ─── Auth ─────────────────────────────────────────────────────────────────────

def browser_auth(server: str) -> dict:
    """Авторизация через браузер (как Claude Code)."""
    import time
    import webbrowser

    print(f"\n{PURPLE}{BOLD}Авторизация Aurora{RESET}")
    print(f"{DIM}Открываю браузер для входа через Telegram...{RESET}\n")

    try:
        # 1. Запросить одноразовый код
        r = httpx.post(f"{server}/api/auth/cli/start", timeout=10)
        data = r.json()
        code = data["code"]
        auth_url = f"{server}{data['url']}"

        # 2. Открыть браузер
        webbrowser.open(auth_url)
        print(f"{DIM}Если браузер не открылся, перейди по ссылке:{RESET}")
        print(f"{GREEN}{auth_url}{RESET}\n")
        print(f"{DIM}Жду авторизацию...{RESET}", end="", flush=True)

        # 3. Поллить пока юзер не залогинится
        for _ in range(60):  # 5 минут макс
            time.sleep(5)
            print(".", end="", flush=True)
            try:
                r = httpx.get(f"{server}/api/auth/cli/poll?code={code}", timeout=5)
                result = r.json()
                if result.get("status") == "ok":
                    print(f"\n\n{GREEN}{BOLD}Авторизация успешна!{RESET}\n")
                    return {
                        "api_key": result["api_key"],
                        "chat_id": result["chat_id"],
                    }
            except Exception:
                pass

        print(f"\n{RED}Время истекло. Попробуй ещё раз.{RESET}\n")
        return {}

    except Exception as e:
        print(f"\n{RED}Ошибка авторизации: {e}{RESET}\n")
        return {}


def ensure_auth(config: dict, server: str) -> dict:
    """Проверяет что пользователь авторизован, если нет — запускает auth flow."""
    if config.get("api_key"):
        return config

    result = browser_auth(server)
    if result.get("api_key"):
        config["api_key"] = result["api_key"]
        config["chat_id"] = result["chat_id"]
        save_config(config)
        return config

    print(f"{RED}Без авторизации Aurora работает в ограниченном режиме.{RESET}\n")
    return config


# ─── Local Project ───────────────────────────────────────────────────────────

class LocalProject:
    """Работа с локальной папкой проекта."""

    def __init__(self, path: str):
        self.path = os.path.abspath(path)
        self.name = os.path.basename(self.path)

    def list_files(self, max_files=50):
        """Список файлов рекурсивно."""
        files = []
        for root, dirs, fnames in os.walk(self.path):
            # Skip hidden dirs and common junk
            dirs[:] = [d for d in dirs if not d.startswith('.') and d not in
                       ('node_modules', '__pycache__', 'venv', '.venv', '.git', 'dist', 'build')]
            for f in fnames:
                if f.startswith('.'):
                    continue
                fp = os.path.join(root, f)
                rel = os.path.relpath(fp, self.path)
                size = os.path.getsize(fp)
                files.append({"name": rel, "size": size})
                if len(files) >= max_files:
                    return files
        return files

    def read_file(self, rel_path: str) -> str:
        """Прочитать файл."""
        fp = os.path.normpath(os.path.join(self.path, rel_path))
        if not fp.startswith(self.path):
            return "[ОШИБКА: выход за пределы проекта]"
        if not os.path.isfile(fp):
            return f"[ОШИБКА: файл не найден: {rel_path}]"
        try:
            with open(fp, "r", encoding="utf-8", errors="replace") as f:
                return f.read()[:50000]
        except Exception as e:
            return f"[ОШИБКА: {e}]"

    def write_file(self, rel_path: str, content: str) -> str:
        """Записать файл."""
        fp = os.path.normpath(os.path.join(self.path, rel_path))
        if not fp.startswith(self.path):
            return "[ОШИБКА: выход за пределы проекта]"
        os.makedirs(os.path.dirname(fp), exist_ok=True)
        with open(fp, "w", encoding="utf-8") as f:
            f.write(content)
        return f"OK: записан {rel_path}"

    def edit_file(self, rel_path: str, old_text: str, new_text: str) -> str:
        """Точечная замена в файле."""
        fp = os.path.normpath(os.path.join(self.path, rel_path))
        if not fp.startswith(self.path):
            return "[ОШИБКА: выход за пределы проекта]"
        if not os.path.isfile(fp):
            return f"[ОШИБКА: файл не найден]"
        with open(fp, "r", encoding="utf-8") as f:
            content = f.read()
        if old_text not in content:
            return "[ОШИБКА: текст для замены не найден]"
        content = content.replace(old_text, new_text, 1)
        with open(fp, "w", encoding="utf-8") as f:
            f.write(content)
        return f"OK: заменено в {rel_path}"

    def tree_string(self) -> str:
        """Дерево файлов как строка для контекста."""
        files = self.list_files()
        lines = [f"{self.name}/"]
        for f in files:
            lines.append(f"  {f['name']} ({f['size']}b)")
        return "\n".join(lines)


# ─── Local Vault ─────────────────────────────────────────────────────────────

class LocalVault:
    """Локальное хранилище: история, память, контекст."""

    def __init__(self, base_dir=None):
        self.base = base_dir or CONFIG_DIR
        self.sessions_dir = os.path.join(self.base, "sessions")
        self.memory_dir = os.path.join(self.base, "memory")
        self.context_file = os.path.join(self.base, "context.json")
        os.makedirs(self.sessions_dir, exist_ok=True)
        os.makedirs(self.memory_dir, exist_ok=True)

    def save_message(self, role, content, session_id=None):
        """Автосохранение каждого сообщения в текущую сессию."""
        from datetime import datetime
        sid = session_id or "default"
        session_file = os.path.join(self.sessions_dir, f"{sid}.jsonl")
        entry = {
            "role": role,
            "content": content[:5000],  # Truncate for storage
            "timestamp": datetime.now().isoformat()
        }
        with open(session_file, "a", encoding="utf-8") as f:
            f.write(json.dumps(entry, ensure_ascii=False) + "\n")

    def get_session_history(self, session_id=None, last_n=20):
        """Получить последние N сообщений сессии."""
        sid = session_id or "default"
        session_file = os.path.join(self.sessions_dir, f"{sid}.jsonl")
        if not os.path.exists(session_file):
            return []
        lines = open(session_file, "r", encoding="utf-8").readlines()
        result = []
        for line in lines[-last_n:]:
            try:
                result.append(json.loads(line))
            except Exception:
                pass
        return result

    def save_context(self, context_data):
        """Сохранить сжатый контекст для восстановления."""
        with open(self.context_file, "w", encoding="utf-8") as f:
            json.dump(context_data, f, ensure_ascii=False, indent=2)

    def load_context(self):
        """Загрузить сохранённый контекст."""
        if os.path.exists(self.context_file):
            try:
                with open(self.context_file, "r", encoding="utf-8") as f:
                    return json.load(f)
            except Exception:
                pass
        return None

    def save_memory(self, key, value):
        """Сохранить факт в локальную память."""
        mem_file = os.path.join(self.memory_dir, "facts.json")
        facts = {}
        if os.path.exists(mem_file):
            try:
                with open(mem_file, "r", encoding="utf-8") as f:
                    facts = json.load(f)
            except Exception:
                pass
        facts[key] = value
        with open(mem_file, "w", encoding="utf-8") as f:
            json.dump(facts, f, ensure_ascii=False, indent=2)

    def get_memory(self):
        """Получить все факты из локальной памяти."""
        mem_file = os.path.join(self.memory_dir, "facts.json")
        if os.path.exists(mem_file):
            try:
                with open(mem_file, "r", encoding="utf-8") as f:
                    return json.load(f)
            except Exception:
                pass
        return {}

    def export_session(self, session_id=None):
        """Экспортировать сессию в Markdown."""
        history = self.get_session_history(session_id, last_n=1000)
        if not history:
            return None
        from datetime import datetime
        lines = [f"# Aurora Session Export", f"> {datetime.now().strftime('%Y-%m-%d %H:%M')}\n"]
        for msg in history:
            role = "User" if msg["role"] == "user" else "Aurora"
            ts = msg.get("timestamp", "")[:16]
            lines.append(f"### {role} ({ts})\n")
            lines.append(msg["content"] + "\n")
        return "\n".join(lines)

    def list_sessions(self):
        """Список локальных сессий."""
        sessions = []
        for f in sorted(os.listdir(self.sessions_dir)):
            if f.endswith(".jsonl"):
                path = os.path.join(self.sessions_dir, f)
                size = os.path.getsize(path)
                lines = sum(1 for _ in open(path, encoding="utf-8"))
                sessions.append({
                    "id": f.replace(".jsonl", ""),
                    "messages": lines,
                    "size": size,
                })
        return sessions


# ─── API ──────────────────────────────────────────────────────────────────────

class AuroraClient:
    def __init__(self, server_url: str, api_key: str = None):
        self.server = server_url.rstrip("/")
        self.api_key = api_key
        self.timeout = httpx.Timeout(120.0, connect=10.0)

    def _headers(self):
        h = {"Content-Type": "application/json"}
        if self.api_key:
            h["Authorization"] = f"Bearer {self.api_key}"
        return h

    def health(self) -> dict:
        r = httpx.get(f"{self.server}/api/health", timeout=self.timeout)
        r.raise_for_status()
        return r.json()

    def projects(self) -> list:
        r = httpx.get(f"{self.server}/api/projects", timeout=self.timeout)
        r.raise_for_status()
        return r.json()

    def create_project(self, name: str) -> dict:
        r = httpx.post(f"{self.server}/api/projects", json={"name": name}, timeout=self.timeout)
        r.raise_for_status()
        return r.json()

    def list_files(self, project: str) -> list:
        r = httpx.get(f"{self.server}/api/files/list", params={"path": project}, timeout=self.timeout)
        r.raise_for_status()
        return r.json()

    def read_file(self, path: str) -> str:
        r = httpx.get(f"{self.server}/api/files/read", params={"path": path}, timeout=self.timeout)
        r.raise_for_status()
        data = r.json()
        if isinstance(data, dict):
            return data.get("content", json.dumps(data, ensure_ascii=False))
        return str(data)

    def send_stream(self, message: str):
        """Стриминг ответа через SSE. Yields текстовые чанки."""
        with httpx.stream(
            "POST",
            f"{self.server}/api/send_stream",
            json={"message": message},
            timeout=self.timeout,
        ) as response:
            response.raise_for_status()
            buffer = ""
            for chunk in response.iter_text():
                buffer += chunk
                while "\n" in buffer:
                    line, buffer = buffer.split("\n", 1)
                    line = line.strip()
                    if not line:
                        continue
                    if line.startswith("data: "):
                        data_str = line[6:]
                        try:
                            data = json.loads(data_str)
                        except json.JSONDecodeError:
                            continue
                        if data.get("done"):
                            return
                        text = data.get("text", "")
                        if text:
                            yield text

    def send(self, message: str, session_id: str = None) -> str:
        """POST запрос с поддержкой tools и сессий."""
        payload = {"message": message}
        if session_id:
            payload["session_id"] = session_id
        r = httpx.post(
            f"{self.server}/api/send",
            json=payload,
            headers=self._headers(),
            timeout=self.timeout,
        )
        r.raise_for_status()
        return r.json().get("response", "")

    def send_image(self, message: str, image_path: str) -> str:
        """Send message with image attachment."""
        with open(image_path, "rb") as f:
            image_bytes = f.read()
        files = {"image": (os.path.basename(image_path), image_bytes, "image/jpeg")}
        data = {"message": message}
        r = httpx.post(
            f"{self.server}/api/vision",
            files=files,
            data=data,
            headers={"Authorization": f"Bearer {self.api_key}"} if self.api_key else {},
            timeout=self.timeout,
        )
        r.raise_for_status()
        return r.json().get("response", "")


# ─── UI ───────────────────────────────────────────────────────────────────────

def print_banner():
    cols = shutil.get_terminal_size((80, 24)).columns
    P = PURPLE
    B = BOLD
    R = RESET
    D = DIM
    art = f"""
{P}{B}  ███╗   ███╗ █████╗ ██████╗      ██████╗  ██████╗ ██████╗     ██╗███╗   ██╗ ██████╗
  ████╗ ████║██╔══██╗██╔══██╗    ██╔════╝ ██╔═══██╗██╔══██╗    ██║████╗  ██║██╔════╝
  ██╔████╔██║███████║██║  ██║    ██║  ███╗██║   ██║██║  ██║    ██║██╔██╗ ██║██║
  ██║╚██╔╝██║██╔══██║██║  ██║    ██║   ██║██║   ██║██║  ██║    ██║██║╚██╗██║██║
  ██║ ╚═╝ ██║██║  ██║██████╔╝    ╚██████╔╝╚██████╔╝██████╔╝    ██║██║ ╚████║╚██████╗
  ╚═╝     ╚═╝╚═╝  ╚═╝╚═════╝      ╚═════╝  ╚═════╝ ╚═════╝     ╚═╝╚═╝  ╚═══╝ ╚═════╝{R}

{D}                          ·  ✦  ·  ★  ·  ✦  ·{R}

{P}                          A U R O R A  AI{R}
{D}                            v{__version__} · 2026{R}
"""
    print(art)
    w = min(cols - 4, 82)
    print(f"{D}  {'─' * w}{R}")
    print(f"{D}  /help — справка  |  /project — проекты  |  /open — локальный проект{R}")
    print()


def print_help():
    help_text = f"""
{BOLD}Команды:{RESET}
  {CYAN}/help{RESET}            — эта справка
  {CYAN}/health{RESET}          — проверка сервера
  {CYAN}/projects{RESET}        — список проектов
  {CYAN}/create NAME{RESET}     — создать проект
  {CYAN}/files NAME{RESET}      — файлы в проекте
  {CYAN}/read path/file{RESET}  — прочитать файл
  {CYAN}/clear{RESET}           — очистить экран
  {CYAN}/project{RESET}         — список проектов, подключение
  {CYAN}/project NAME{RESET}   — подключиться к проекту
  {CYAN}/project off{RESET}    — отключиться от проекта
  {CYAN}/open PATH{RESET}      — открыть локальную папку
  {CYAN}/close{RESET}           — закрыть локальный проект
  {CYAN}/files{RESET}           — файлы текущего проекта
  {CYAN}/session{RESET}         — список сессий / создать / подключиться
  {CYAN}/session new{RESET}    — создать сессию (для синхронизации с TG)
  {CYAN}/session ID{RESET}     — подключиться к сессии
  {CYAN}/session off{RESET}    — отключиться
  {CYAN}/approve{RESET}         — переключить режим подтверждений
  {CYAN}/server URL{RESET}      — сменить сервер
  {CYAN}/login{RESET}           — авторизация через браузер
  {CYAN}/logout{RESET}          — выйти из аккаунта
  {CYAN}/quit{RESET}            — выход

{BOLD}Локальное хранилище:{RESET}
  {CYAN}/memory{RESET}           — показать локальную память
  {CYAN}/memory key=val{RESET}  — сохранить факт
  {CYAN}/compress{RESET}         — сжать контекст сессии
  {CYAN}/export{RESET}           — экспортировать сессию в Markdown
  {CYAN}/history{RESET}          — последние 10 сообщений
  {CYAN}/sessions{RESET}         — список локальных сессий

{BOLD}Режимы подтверждения:{RESET}
  По умолчанию Aurora спрашивает перед изменением файлов.
  /approve auto  — применять все изменения без вопросов
  /approve ask   — спрашивать каждый раз (по умолчанию)

{BOLD}Использование:{RESET}
  aurora               — интерактивный режим
  aurora "сообщение"   — одиночный запрос
"""
    print(help_text)


def prompt_input(config: dict = None) -> str:
    try:
        lp = config.get("_local_proj_obj") if config else None
        proj = config.get("active_project") if config else None
        sid = config.get("session_id") if config else None
        prefix = ""
        if lp:
            prefix = f"[{lp.name}]"
        elif proj:
            prefix = f"[{proj}]"
        if sid:
            prefix = f"[S:{sid[:6]}]{prefix}"
        msg = input(f"{GREEN}{BOLD}{prefix}> {RESET}")
        return msg.strip()
    except (EOFError, KeyboardInterrupt):
        print()
        return "/quit"


# ─── Обработка команд ─────────────────────────────────────────────────────────

def handle_command(cmd: str, client: AuroraClient, config: dict) -> bool:
    """Обрабатывает слеш-команды. Возвращает False если надо выйти."""
    parts = cmd.split(maxsplit=1)
    command = parts[0].lower()
    arg = parts[1] if len(parts) > 1 else ""

    if command == "/quit" or command == "/exit":
        print(f"{DIM}Пока!{RESET}")
        return False

    elif command == "/help":
        print_help()

    elif command == "/clear":
        os.system("cls" if os.name == "nt" else "clear")
        print_banner()

    elif command == "/health":
        try:
            data = client.health()
            print(f"{GREEN}Сервер OK{RESET}: {json.dumps(data, ensure_ascii=False, indent=2)}")
        except Exception as e:
            print(f"{RED}Ошибка{RESET}: {e}")

    elif command == "/projects":
        try:
            data = client.projects()
            if isinstance(data, list):
                if not data:
                    print(f"{DIM}Проектов нет{RESET}")
                for p in data:
                    if isinstance(p, dict):
                        print(f"  {CYAN}{p.get('name', p)}{RESET}")
                    else:
                        print(f"  {CYAN}{p}{RESET}")
            else:
                print(json.dumps(data, ensure_ascii=False, indent=2))
        except Exception as e:
            print(f"{RED}Ошибка{RESET}: {e}")

    elif command == "/create":
        if not arg:
            print(f"{RED}Укажи имя: /create NAME{RESET}")
        else:
            try:
                data = client.create_project(arg)
                print(f"{GREEN}Создан{RESET}: {json.dumps(data, ensure_ascii=False)}")
            except Exception as e:
                print(f"{RED}Ошибка{RESET}: {e}")

    elif command == "/files":
        if not arg:
            print(f"{RED}Укажи проект: /files NAME{RESET}")
        else:
            try:
                data = client.list_files(arg)
                if isinstance(data, list):
                    if not data:
                        print(f"{DIM}Файлов нет{RESET}")
                    for f in data:
                        if isinstance(f, dict):
                            print(f"  {f.get('name', f)}")
                        else:
                            print(f"  {f}")
                else:
                    print(json.dumps(data, ensure_ascii=False, indent=2))
            except Exception as e:
                print(f"{RED}Ошибка{RESET}: {e}")

    elif command == "/read":
        if not arg:
            print(f"{RED}Укажи путь: /read project/file{RESET}")
        else:
            try:
                content = client.read_file(arg)
                lines = content.split("\n")
                for i, line in enumerate(lines, 1):
                    print(f"{DIM}{i:4}{RESET}  {line}")
            except Exception as e:
                print(f"{RED}Ошибка{RESET}: {e}")

    elif command == "/server":
        if not arg:
            print(f"Текущий сервер: {CYAN}{client.server}{RESET}")
        else:
            url = arg if arg.startswith("http") else f"http://{arg}"
            client.server = url.rstrip("/")
            config["server"] = client.server
            save_config(config)
            print(f"{GREEN}Сервер изменён{RESET}: {client.server}")

    elif command == "/login":
        result = browser_auth(client.server)
        if result.get("api_key"):
            config["api_key"] = result["api_key"]
            config["chat_id"] = result["chat_id"]
            save_config(config)
            client.api_key = result["api_key"]
            print(f"{GREEN}Авторизация обновлена{RESET}")

    elif command == "/logout":
        config.pop("api_key", None)
        config.pop("chat_id", None)
        save_config(config)
        client.api_key = None
        print(f"{GREEN}Вышел из аккаунта{RESET}")

    elif command == "/project":
        if arg.lower() == "off":
            config.pop("active_project", None)
            print(f"{GREEN}Отключена от проекта{RESET}")
        elif arg:
            config["active_project"] = arg
            print(f"{GREEN}Подключена к проекту: {PURPLE}{arg}{RESET}")
        else:
            # Show list
            try:
                data = client.projects()
                projects = data.get("projects", [])
                if not projects:
                    print(f"{DIM}Нет проектов. Попроси Aurora создать.{RESET}")
                else:
                    print(f"\n{CYAN}Проекты:{RESET}")
                    for i, p in enumerate(projects, 1):
                        marker = f" {GREEN}●{RESET}" if config.get("active_project") == p["name"] else ""
                        print(f"  {BOLD}{i}.{RESET} {p['name']} {DIM}({p.get('files', 0)} файлов){RESET}{marker}")
                    print(f"\n{DIM}Введи номер или имя:{RESET} ", end="")
                    choice = input().strip()
                    if choice.isdigit():
                        idx = int(choice) - 1
                        if 0 <= idx < len(projects):
                            config["active_project"] = projects[idx]["name"]
                            print(f"{GREEN}Подключена к: {PURPLE}{projects[idx]['name']}{RESET}")
                    elif choice:
                        config["active_project"] = choice
                        print(f"{GREEN}Подключена к: {PURPLE}{choice}{RESET}")
            except Exception as e:
                print(f"{RED}Ошибка: {e}{RESET}")

    elif command == "/open":
        path = arg or "."
        path = os.path.abspath(path)
        if not os.path.isdir(path):
            print(f"{RED}Папка не найдена: {path}{RESET}")
        else:
            config["local_project"] = path
            lp = LocalProject(path)
            config["_local_proj_obj"] = lp
            files = lp.list_files()
            print(f"\n{GREEN}Открыт локальный проект: {PURPLE}{lp.name}{RESET}")
            print(f"{DIM}{path}{RESET}")
            print(f"{DIM}{len(files)} файлов{RESET}\n")
            for f in files[:15]:
                print(f"  {DIM}📄 {f['name']} ({f['size']}b){RESET}")
            if len(files) > 15:
                print(f"  {DIM}... и ещё {len(files)-15}{RESET}")
            print()

    elif command == "/close":
        config.pop("local_project", None)
        config.pop("_local_proj_obj", None)
        print(f"{GREEN}Локальный проект закрыт{RESET}")

    elif command == "/session":
        if arg.lower() == "new" or arg.lower() == "create":
            try:
                name = input(f"{DIM}Название сессии: {RESET}").strip() or ""
                r = httpx.post(f"{client.server}/api/session/create",
                              json={"name": name}, headers=client._headers(), timeout=10)
                data = r.json()
                sid = data.get("session_id", "")
                config["session_id"] = sid
                print(f"\n{GREEN}Сессия создана: {PURPLE}{sid}{RESET}")
                print(f"{DIM}Для подключения в TG: /session {sid}{RESET}\n")
            except Exception as e:
                print(f"{RED}Ошибка: {e}{RESET}")
        elif arg.lower() == "off" or arg.lower() == "end":
            config.pop("session_id", None)
            print(f"{GREEN}Отключена от сессии{RESET}")
        elif arg:
            config["session_id"] = arg
            print(f"{GREEN}Подключена к сессии: {PURPLE}{arg}{RESET}")
        else:
            try:
                r = httpx.get(f"{client.server}/api/sessions", headers=client._headers(), timeout=10)
                data = r.json()
                sessions = data.get("sessions", [])
                current = config.get("session_id")
                if not sessions:
                    print(f"{DIM}Нет сессий. /session new — создать{RESET}")
                else:
                    print(f"\n{CYAN}Сессии:{RESET}")
                    for s in sessions:
                        marker = f" {GREEN}● активна{RESET}" if s["id"] == current else ""
                        print(f"  {BOLD}{s['id']}{RESET} — {s.get('name', '')}{marker}")
                    print(f"\n{DIM}/session ID — подключиться | /session new — создать{RESET}\n")
            except Exception as e:
                print(f"{RED}Ошибка: {e}{RESET}")

    elif command == "/approve":
        global _approve_mode
        if arg.lower() in ("auto", "all", "да"):
            _approve_mode = APPROVE_AUTO
            print(f"{GREEN}Режим: автоподтверждение{RESET} — файлы применяются без вопросов")
        elif arg.lower() in ("ask", "manual", "нет"):
            _approve_mode = APPROVE_ASK
            print(f"{GREEN}Режим: ручное подтверждение{RESET} — спрашиваю перед каждым изменением")
        else:
            current = "автоподтверждение" if _approve_mode == APPROVE_AUTO else "ручное подтверждение"
            print(f"Текущий режим: {CYAN}{current}{RESET}")
            print(f"{DIM}/approve auto — не спрашивать{RESET}")
            print(f"{DIM}/approve ask  — спрашивать каждый раз{RESET}")

    elif command == "/files":
        # Local project files
        lp = config.get("_local_proj_obj")
        if lp and not arg:
            files = lp.list_files()
            print(f"\n{CYAN}{lp.name}/{RESET}")
            for f in files:
                print(f"  📄 {f['name']} {DIM}{f['size']}b{RESET}")
            print()
            return True

        proj = arg or config.get("active_project")
        if not proj:
            print(f"{RED}Укажи проект: /files NAME или /project для подключения{RESET}")
        else:
            try:
                data = client.files(proj)
                files = data.get("files", [])
                if not files:
                    print(f"{DIM}Проект пуст{RESET}")
                else:
                    print(f"\n{CYAN}{proj}/{RESET}")
                    for f in files:
                        icon = "📁" if f.get("is_dir") else "📄"
                        size = f" {DIM}{f.get('size', 0)}b{RESET}" if not f.get("is_dir") else ""
                        print(f"  {icon} {f['name']}{size}")
                print()
            except Exception as e:
                print(f"{RED}Ошибка: {e}{RESET}")

    elif command == "/compress":
        vault = config.get("_vault")
        if not vault:
            print(f"{RED}Vault не инициализирован{RESET}")
            return True
        sid = config.get("session_id", "default")
        history = vault.get_session_history(sid, last_n=50)
        if not history:
            print(f"{DIM}Нет истории для сжатия{RESET}")
            return True
        # Send to Aurora for summarization
        msgs_text = "\n".join(f"{m['role']}: {m['content'][:200]}" for m in history[-20:])
        summary_prompt = f"Сожми этот диалог в краткое резюме (3-5 предложений), сохрани ключевые факты и решения:\n\n{msgs_text}"
        try:
            print(f"{DIM}Сжимаю контекст...{RESET}")
            summary = client.send(summary_prompt)
            vault.save_context({"summary": summary, "session_id": sid, "messages_compressed": len(history)})
            print(f"\n{GREEN}Контекст сжат и сохранён{RESET}")
            print(f"{DIM}{summary[:300]}...{RESET}\n")
        except Exception as e:
            print(f"{RED}Ошибка: {e}{RESET}")

    elif command == "/memory":
        vault = config.get("_vault")
        if not vault:
            print(f"{RED}Vault не инициализирован{RESET}")
            return True
        if arg:
            # Save a fact
            if "=" in arg:
                key, val = arg.split("=", 1)
                vault.save_memory(key.strip(), val.strip())
                print(f"{GREEN}Сохранено: {key.strip()} = {val.strip()}{RESET}")
            else:
                print(f"{DIM}Формат: /memory ключ = значение{RESET}")
        else:
            facts = vault.get_memory()
            if not facts:
                print(f"{DIM}Память пуста. /memory ключ = значение — сохранить факт{RESET}")
            else:
                print(f"\n{CYAN}Локальная память:{RESET}")
                for k, v in facts.items():
                    print(f"  {BOLD}{k}{RESET}: {v}")
                print()

    elif command == "/export":
        vault = config.get("_vault")
        if not vault:
            print(f"{RED}Vault не инициализирован{RESET}")
            return True
        sid = arg or config.get("session_id", "default")
        md = vault.export_session(sid)
        if not md:
            print(f"{DIM}Нет истории для экспорта{RESET}")
            return True
        export_file = os.path.join(CONFIG_DIR, f"export_{sid}.md")
        with open(export_file, "w", encoding="utf-8") as f:
            f.write(md)
        print(f"{GREEN}Экспортировано: {export_file}{RESET}")

    elif command == "/history":
        vault = config.get("_vault")
        if not vault:
            print(f"{RED}Vault не инициализирован{RESET}")
            return True
        sid = config.get("session_id", "default")
        history = vault.get_session_history(sid, last_n=10)
        if not history:
            print(f"{DIM}Нет истории{RESET}")
        else:
            print(f"\n{CYAN}Последние сообщения:{RESET}")
            for msg in history:
                role = f"{GREEN}Вы{RESET}" if msg["role"] == "user" else f"{PURPLE}Aurora{RESET}"
                ts = msg.get("timestamp", "")[:16]
                content = msg["content"][:100].replace("\n", " ")
                print(f"  {DIM}{ts}{RESET} {role}: {content}")
            print()

    elif command == "/sessions":
        vault = config.get("_vault")
        if not vault:
            print(f"{RED}Vault не инициализирован{RESET}")
            return True
        sessions = vault.list_sessions()
        if not sessions:
            print(f"{DIM}Нет локальных сессий{RESET}")
        else:
            print(f"\n{CYAN}Локальные сессии:{RESET}")
            for s in sessions:
                current = " ● " if s["id"] == config.get("session_id", "default") else "   "
                print(f"  {GREEN}{current}{RESET}{s['id']} — {s['messages']} сообщений ({s['size']}b)")
            print()

    else:
        print(f"{RED}Неизвестная команда{RESET}: {command}")
        print(f"{DIM}Введи /help для справки{RESET}")

    return True


# ─── Отправка сообщения ───────────────────────────────────────────────────────

def _handle_local_request(message: str, config: dict) -> str:
    """Проверяет упоминание локальных путей и автоматически читает файлы/папки."""
    import re

    # 1. Ищем явные пути: C:/..., ~/..., ./...
    paths = re.findall(r'(?:[A-Za-z]:[/\\][^\s,\"]+|~/[^\s,\"]+|\./[^\s,\"]+)', message)

    # 2. Ищем упоминания "папка X на рабочем столе" / "папку brain-work"
    desktop = os.path.expanduser("~/Desktop")
    folder_mentions = re.findall(r'(?:папк[уае]\s+|folder\s+|директори[юя]\s+)(\S+)', message, re.IGNORECASE)
    for name in folder_mentions:
        name = name.strip('.,!?"\' ')
        # Попробовать на рабочем столе
        candidate = os.path.join(desktop, name)
        if os.path.isdir(candidate):
            paths.append(candidate)
        # Попробовать в текущей папке
        elif os.path.isdir(name):
            paths.append(os.path.abspath(name))

    # 3. Ищем "на рабочем столе есть X" / "рабочем столе X"
    desk_mentions = re.findall(r'(?:рабоч\w+\s+стол\w*\s+(?:есть\s+|)?)(\S+)', message, re.IGNORECASE)
    for name in desk_mentions:
        name = name.strip('.,!?"\' ')
        if name.lower() in ('и', 'а', 'что', 'там', 'папку', 'файл', 'есть'):
            continue
        candidate = os.path.join(desktop, name)
        if os.path.isdir(candidate):
            paths.append(candidate)
        elif os.path.isfile(candidate):
            paths.append(candidate)

    # Убрать дубли
    seen = set()
    unique_paths = []
    for p in paths:
        p = os.path.abspath(os.path.expanduser(p))
        if p not in seen:
            seen.add(p)
            unique_paths.append(p)

    context_parts = []
    for p in unique_paths:
        if os.path.isdir(p):
            lp = LocalProject(p)
            tree = lp.tree_string()
            context_parts.append(f"[Содержимое папки: {p}]\n{tree}")
        elif os.path.isfile(p):
            try:
                with open(p, "r", encoding="utf-8", errors="replace") as f:
                    content = f.read()[:30000]
                context_parts.append(f"[Файл: {p}]\n```\n{content}\n```")
            except Exception as e:
                context_parts.append(f"[Ошибка чтения {p}: {e}]")

    if context_parts:
        return message + "\n\n" + "\n\n".join(context_parts)
    return message


def send_message(message: str, client: AuroraClient, config: dict = None):
    """Отправляет сообщение и печатает ответ."""
    import re

    original_message = message

    # Контекст проекта
    proj = config.get("active_project") if config else None
    lp = config.get("_local_proj_obj") if config else None

    if lp and not message.startswith("/"):
        tree = lp.tree_string()
        message = f"[Локальный проект: {lp.name}]\nСтруктура:\n{tree}\n\nЗапрос: {message}"
    elif proj and not message.startswith("/"):
        message = f"[Проект: {proj}] {message}"
    else:
        # Автоматически подхватываем локальные пути из сообщения
        message = _handle_local_request(message, config or {})

    # Анимация "думает"
    import threading
    stop_spinner = threading.Event()
    def spinner():
        frames = ["⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"]
        i = 0
        while not stop_spinner.is_set():
            print(f"\r{PURPLE}  {frames[i % len(frames)]} Aurora думает...{RESET}  ", end="", flush=True)
            i += 1
            stop_spinner.wait(0.1)
        print(f"\r{' ' * 40}\r", end="", flush=True)

    spin_thread = threading.Thread(target=spinner, daemon=True)
    spin_thread.start()

    try:
        sid = config.get("session_id") if config else None

        # Check for image paths in message
        import re as _re
        image_exts = ('.png', '.jpg', '.jpeg', '.gif', '.bmp', '.webp', '.tiff')
        image_paths = _re.findall(r'(?:[A-Za-z]:[/\\][^\s,\"]+|~/[^\s,\"]+|\./[^\s,\"]+|/[^\s,\"]+)', message)
        image_path = None
        for p in image_paths:
            p = os.path.abspath(os.path.expanduser(p))
            if os.path.isfile(p) and p.lower().endswith(image_exts):
                image_path = p
                break

        if image_path:
            # Remove image path from message text
            clean_msg = message
            for p in image_paths:
                clean_msg = clean_msg.replace(p, "").strip()
            if not clean_msg:
                clean_msg = "Что на этом изображении?"
            resp = client.send_image(clean_msg, image_path)
        else:
            resp = client.send(message, session_id=sid)

        vault = config.get("_vault") if config else None
        if vault:
            vault.save_message("user", original_message, sid)
            vault.save_message("assistant", resp, sid)

        stop_spinner.set()
        spin_thread.join(timeout=1)

        # Filter raw tool call tags
        resp = re.sub(r'<\|tool_call>.*?<tool_call\|>', '', resp, flags=re.DOTALL).strip()

        # Handle local_shell — execute commands on user's PC
        local_exec_rounds = 0
        while "[LOCAL_EXEC_PENDING]" in resp and local_exec_rounds < 5:
            local_exec_rounds += 1
            # Extract command and reason
            match = re.search(r'\[LOCAL_EXEC_PENDING\]\s*command=(.*?)\s*reason=(.*?)(?:\n|$)', resp)
            if not match:
                break
            cmd = match.group(1).strip()
            reason = match.group(2).strip()

            # Clean response — show only text before the marker
            clean_resp = resp[:resp.index("[LOCAL_EXEC_PENDING]")].strip()
            if clean_resp:
                print(f"{PURPLE}{BOLD}Aurora:{RESET} {clean_resp}")

            # Ask user permission
            print(f"\n  {YELLOW}Aurora хочет выполнить команду на вашем ПК:{RESET}")
            print(f"  {CYAN}$ {cmd}{RESET}")
            if reason:
                print(f"  {DIM}Причина: {reason}{RESET}")
            answer = input(f"  {GREEN}Разрешить? [y/n]: {RESET}").strip().lower()

            if answer in ('y', 'yes', 'да', ''):
                print(f"  {DIM}Выполняю...{RESET}")
                try:
                    import subprocess
                    # Detect OS and use appropriate shell
                    if sys.platform == "win32":
                        result = subprocess.run(
                            ["powershell", "-Command", cmd],
                            capture_output=True, text=True, timeout=60
                        )
                    else:
                        result = subprocess.run(
                            cmd, shell=True,
                            capture_output=True, text=True, timeout=60
                        )
                    output = result.stdout.strip()
                    if result.stderr.strip():
                        output += "\n[STDERR] " + result.stderr.strip()
                    if not output:
                        output = "(команда выполнена, вывод пустой)"
                    print(f"  {GREEN}Результат:{RESET}\n  {output[:500]}")

                    # Send result back to Aurora for analysis
                    followup = f"Результат команды `{cmd}`:\n```\n{output[:2000]}\n```\nПроанализируй результат и продолжай решать задачу."
                    print(f"\n{PURPLE}  ⠋ Aurora анализирует результат...{RESET}")
                    resp = client.send(followup, session_id=sid)
                    resp = re.sub(r'<\|tool_call>.*?<tool_call\|>', '', resp, flags=re.DOTALL).strip()
                except subprocess.TimeoutExpired:
                    print(f"  {RED}Таймаут (60 сек){RESET}")
                    resp = ""
                    break
                except Exception as e:
                    print(f"  {RED}Ошибка: {e}{RESET}")
                    resp = ""
                    break
            else:
                print(f"  {DIM}Отклонено{RESET}")
                resp = client.send("Пользователь отклонил выполнение команды. Предложи альтернативное решение.", session_id=sid)
                resp = re.sub(r'<\|tool_call>.*?<tool_call\|>', '', resp, flags=re.DOTALL).strip()

        print(f"{PURPLE}{BOLD}Aurora:{RESET} ", end="", flush=True)

        # Если локальный проект — проверяем есть ли команды на создание/изменение файлов
        if lp and resp:
            # Ищем code blocks с именами файлов
            file_blocks = re.findall(r'`([^`]+\.\w+)`[:\s]*\n```\w*\n([\s\S]*?)```', resp)
            if file_blocks:
                for fname, content in file_blocks:
                    fname = fname.strip().lstrip('/')
                    if _approve_mode == APPROVE_AUTO:
                        result = lp.write_file(fname, content)
                        print(f"\n  {GREEN}✓ {fname}{RESET} {DIM}{result}{RESET}")
                    else:
                        print(f"\n  {CYAN}Файл: {fname}{RESET} ({len(content)} символов)")
                        print(f"  {DIM}Превью: {content[:80].replace(chr(10), ' ')}...{RESET}")
                        answer = input(f"  {GREEN}Применить? [y/n/a(все)]: {RESET}")
                        if answer.lower() in ('a', 'all', 'все'):
                            _approve_mode = APPROVE_AUTO
                            result = lp.write_file(fname, content)
                            print(f"  {GREEN}✓ {result}{RESET}")
                            print(f"  {DIM}Режим: автоподтверждение{RESET}")
                        elif answer.lower() in ('y', 'да', ''):
                            result = lp.write_file(fname, content)
                            print(f"  {GREEN}✓ {result}{RESET}")
                        else:
                            print(f"  {DIM}Пропущено{RESET}")

        if resp:
            print(f"{resp}\n")
        else:
            print(f"{DIM}(выполняю действие...){RESET}\n")
    except KeyboardInterrupt:
        stop_spinner.set()
        print(f"\n{DIM}(прервано){RESET}\n")
    except Exception as e:
        stop_spinner.set()
        print(f"\n{RED}Ошибка{RESET}: {e}\n")


# ─── Main ─────────────────────────────────────────────────────────────────────

def main():
    # Сигнал для ctrl+c
    signal.signal(signal.SIGINT, lambda *_: None)

    config = load_config()
    server = config.get("server", DEFAULT_SERVER)

    # Auth flow — если нет API ключа, авторизуемся через браузер
    config = ensure_auth(config, server)

    client = AuroraClient(server, api_key=config.get("api_key"))

    vault = LocalVault()
    config["_vault"] = vault

    # Single-shot mode
    if len(sys.argv) > 1:
        message = " ".join(sys.argv[1:])
        if message.startswith("/"):
            handle_command(message, client, config)
        else:
            send_message(message, client)
        return

    # Interactive REPL
    print_banner()

    while True:
        try:
            msg = prompt_input(config)
        except KeyboardInterrupt:
            print(f"\n{DIM}Пока!{RESET}")
            break

        if not msg:
            continue

        if msg.startswith("/"):
            if not handle_command(msg, client, config):
                break
        else:
            send_message(msg, client, config)


if __name__ == "__main__":
    main()
