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

__version__ = "0.1.0"

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

    def send(self, message: str) -> str:
        """POST запрос с поддержкой tools."""
        r = httpx.post(
            f"{self.server}/api/send",
            json={"message": message},
            headers=self._headers(),
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
  {CYAN}/files{RESET}           — файлы текущего проекта
  {CYAN}/server URL{RESET}      — сменить сервер
  {CYAN}/login{RESET}           — авторизация через браузер
  {CYAN}/logout{RESET}          — выйти из аккаунта
  {CYAN}/quit{RESET}            — выход

{BOLD}Использование:{RESET}
  aurora               — интерактивный режим
  aurora "сообщение"   — одиночный запрос
"""
    print(help_text)


def prompt_input(config: dict = None) -> str:
    try:
        lp = config.get("_local_proj_obj") if config else None
        proj = config.get("active_project") if config else None
        if lp:
            msg = input(f"{GREEN}{BOLD}[{lp.name}]> {RESET}")
        elif proj:
            msg = input(f"{GREEN}{BOLD}[{proj}]> {RESET}")
        else:
            msg = input(f"{GREEN}{BOLD}> {RESET}")
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

    else:
        print(f"{RED}Неизвестная команда{RESET}: {command}")
        print(f"{DIM}Введи /help для справки{RESET}")

    return True


# ─── Отправка сообщения ───────────────────────────────────────────────────────

def send_message(message: str, client: AuroraClient, config: dict = None):
    """Отправляет сообщение и печатает ответ."""
    import re

    # Контекст проекта
    proj = config.get("active_project") if config else None
    lp = config.get("_local_proj_obj") if config else None

    if lp and not message.startswith("/"):
        # Локальный проект — добавляем дерево файлов в контекст
        tree = lp.tree_string()
        message = f"[Локальный проект: {lp.name}]\nСтруктура:\n{tree}\n\nЗапрос: {message}"
    elif proj and not message.startswith("/"):
        message = f"[Проект: {proj}] {message}"

    print(f"\n{PURPLE}{BOLD}Aurora:{RESET} ", end="", flush=True)
    try:
        resp = client.send(message)
        # Filter raw tool call tags
        resp = re.sub(r'<\|tool_call>.*?<tool_call\|>', '', resp, flags=re.DOTALL).strip()

        # Если локальный проект — проверяем есть ли команды на создание/изменение файлов
        if lp and resp:
            # Ищем code blocks с именами файлов
            file_blocks = re.findall(r'`([^`]+\.\w+)`[:\s]*\n```\w*\n([\s\S]*?)```', resp)
            if file_blocks:
                for fname, content in file_blocks:
                    fname = fname.strip().lstrip('/')
                    answer = input(f"\n{GREEN}Создать/обновить {fname}? [y/n]: {RESET}")
                    if answer.lower() in ('y', 'да', ''):
                        result = lp.write_file(fname, content)
                        print(f"  {DIM}{result}{RESET}")

        if resp:
            print(f"{resp}\n")
        else:
            print(f"{DIM}(выполняю действие...){RESET}\n")
    except KeyboardInterrupt:
        print(f"\n{DIM}(прервано){RESET}\n")
    except Exception as e:
        print(f"\n{RED}Ошибка{RESET}: {e}\n")
    except Exception as e:
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
