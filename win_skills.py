import chromadb

client = chromadb.PersistentClient(path="/data/mempalace/aurora/")

# === Collection 1: windows_troubleshooting ===
ts = client.get_or_create_collection("windows_troubleshooting")

ts_skills = [
    {
        "id": "win_ts_audio",
        "document": "Диагностика звука Windows:\n1. Проверка устройств: Get-AudioDevice (AudioDeviceCmdlets), Settings > Sound\n2. Службы: Get-Service Audiosrv, AudioEndpointBuilder — обе должны Running\n3. Restart-Service Audiosrv -Force\n4. Драйверы: devmgmt.msc > Sound > Update/Rollback driver\n5. Troubleshooter: msdt.exe /id AudioPlaybackDiagnostic\n6. Проверка уровней: sndvol, mixer settings\n7. PowerShell: Get-PnpDevice -Class AudioEndpoint | Where Status -eq Error",
        "metadata": {"tags": "audio,sound,audiosrv,AudioEndpointBuilder,driver,troubleshoot", "category": "windows_troubleshooting", "os": "windows"}
    },
    {
        "id": "win_ts_network",
        "document": "Диагностика сети Windows:\n1. ipconfig /all — адреса, DNS, DHCP, шлюз\n2. ping 8.8.8.8 — проверка интернета; ping localhost — loopback\n3. nslookup domain.com — проверка DNS резолва\n4. tracert domain.com — трассировка маршрута\n5. netsh winsock reset — сброс Winsock каталога\n6. ipconfig /flushdns — очистка DNS кэша\n7. ipconfig /release && ipconfig /renew — обновление DHCP\n8. netsh int ip reset — полный сброс TCP/IP стека\n9. Get-NetAdapter | Where Status -eq Up — активные адаптеры\n10. Test-NetConnection -ComputerName google.com -Port 443",
        "metadata": {"tags": "network,ipconfig,ping,dns,winsock,tracert,netsh", "category": "windows_troubleshooting", "os": "windows"}
    },
    {
        "id": "win_ts_disk",
        "document": "Диагностика диска Windows:\n1. chkdsk C: /f /r — проверка и ремонт файловой системы (требует перезагрузку)\n2. sfc /scannow — проверка целостности системных файлов\n3. DISM /Online /Cleanup-Image /RestoreHealth — восстановление образа\n4. cleanmgr — очистка диска (GUI), cleanmgr /sageset:1 для настройки\n5. Get-PhysicalDisk | Get-StorageReliabilityCounter — SMART данные\n6. wmic diskdrive get status — быстрая проверка здоровья\n7. Optimize-Volume -DriveLetter C -Defrag — дефрагментация HDD\n8. Optimize-Volume -DriveLetter C -ReTrim — TRIM для SSD\n9. Get-Volume — список томов и свободное место",
        "metadata": {"tags": "disk,chkdsk,sfc,DISM,SMART,cleanup,defrag", "category": "windows_troubleshooting", "os": "windows"}
    },
    {
        "id": "win_ts_updates",
        "document": "Обновления Windows:\n1. Settings > Windows Update > Check for updates\n2. PowerShell: Install-Module PSWindowsUpdate; Get-WindowsUpdate\n3. Install-WindowsUpdate -AcceptAll -AutoReboot\n4. Откат: Settings > Recovery > Go back (10 дней)\n5. Удаление обновления: wusa /uninstall /kb:NNNNNNN\n6. Очистка кэша: net stop wuauserv, del C:\\Windows\\SoftwareDistribution\\*, net start wuauserv\n7. DISM /Online /Cleanup-Image /StartComponentCleanup — очистка старых версий\n8. Get-HotFix | Sort InstalledOn -Descending — список установленных\n9. UsoClient StartScan — принудительный запуск проверки",
        "metadata": {"tags": "update,WindowsUpdate,hotfix,wusa,DISM,patch", "category": "windows_troubleshooting", "os": "windows"}
    },
    {
        "id": "win_ts_services",
        "document": "Управление службами Windows:\n1. Get-Service — список всех служб\n2. Get-Service -Name wuauserv — проверка конкретной службы\n3. Start-Service -Name Spooler; Stop-Service; Restart-Service\n4. Set-Service -Name ssh-agent -StartupType Automatic\n5. sc.exe query type=service state=all — через sc.exe\n6. sc.exe config ServiceName start=auto — изменение типа запуска\n7. services.msc — GUI менеджер служб\n8. Get-Service | Where Status -eq Stopped | Where StartType -eq Automatic — авто-службы что не запущены\n9. Get-WmiObject Win32_Service | Where State -eq Stopped | Select Name,StartMode",
        "metadata": {"tags": "service,Get-Service,sc.exe,Start-Service,startup,services.msc", "category": "windows_troubleshooting", "os": "windows"}
    },
    {
        "id": "win_ts_processes",
        "document": "Управление процессами Windows:\n1. Get-Process — список процессов\n2. Get-Process | Sort WorkingSet -Descending | Select -First 10 — топ по RAM\n3. Get-Process | Sort CPU -Descending | Select -First 10 — топ по CPU\n4. Stop-Process -Name notepad -Force; Stop-Process -Id 1234\n5. taskkill /F /IM process.exe; taskkill /F /PID 1234\n6. tasklist /v — подробный список с заголовками окон\n7. Утечки памяти: Get-Process | Where {$_.WorkingSet -gt 1GB}\n8. Wait-Process -Name setup — ожидание завершения\n9. Start-Process -FilePath cmd -Verb RunAs — запуск от админа\n10. Диспетчер: taskmgr, resmon (Resource Monitor)",
        "metadata": {"tags": "process,taskkill,Get-Process,memory,CPU,taskmgr", "category": "windows_troubleshooting", "os": "windows"}
    },
    {
        "id": "win_ts_registry",
        "document": "Реестр Windows:\n1. regedit — GUI редактор реестра\n2. Get-ItemProperty -Path HKLM:\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\n3. Set-ItemProperty -Path HKCU:\\... -Name Key -Value Data\n4. New-Item -Path HKCU:\\Software\\MyApp — создание ключа\n5. Remove-Item -Path HKCU:\\Software\\MyApp -Recurse\n6. reg export HKCU\\Software\\MyApp backup.reg — экспорт/бэкап\n7. reg import backup.reg — импорт/восстановление\n8. Важные ветки: HKLM\\SYSTEM\\CurrentControlSet, HKLM\\SOFTWARE, HKCU\\Software\n9. reg query HKLM\\SOFTWARE\\Microsoft /s /f keyword — поиск",
        "metadata": {"tags": "registry,regedit,HKLM,HKCU,Get-ItemProperty,reg", "category": "windows_troubleshooting", "os": "windows"}
    },
    {
        "id": "win_ts_startup",
        "document": "Автозагрузка Windows:\n1. Task Manager > Startup tab — управление автозагрузкой\n2. msconfig > Startup — классический способ\n3. Папки: shell:startup (пользователь), shell:common startup (все)\n4. Реестр: HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run\n5. HKLM\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Run\n6. Get-CimInstance Win32_StartupCommand | Select Name,Command,Location\n7. Task Scheduler: taskschd.msc — планировщик задач\n8. Get-ScheduledTask | Where State -eq Ready — активные задачи\n9. Disable-ScheduledTask -TaskName TaskName\n10. autoruns.exe (Sysinternals) — самый полный просмотр автозагрузки",
        "metadata": {"tags": "startup,autorun,msconfig,TaskScheduler,boot,login", "category": "windows_troubleshooting", "os": "windows"}
    },
    {
        "id": "win_ts_drivers",
        "document": "Драйверы Windows:\n1. devmgmt.msc — Device Manager GUI\n2. Get-PnpDevice — список устройств и статус\n3. Get-PnpDevice | Where Status -eq Error — проблемные устройства\n4. pnputil /enum-drivers — список установленных драйверов\n5. pnputil /add-driver driver.inf /install — установка драйвера\n6. pnputil /delete-driver oemNN.inf /force — удаление драйвера\n7. Откат: Device Manager > Properties > Driver > Roll Back\n8. driverquery /v — подробный список драйверов\n9. sigverif — проверка подписей драйверов\n10. DISM /Online /Export-Driver /Destination:C:\\DriversBackup — бэкап всех",
        "metadata": {"tags": "driver,DeviceManager,pnputil,PnpDevice,hardware,update", "category": "windows_troubleshooting", "os": "windows"}
    },
    {
        "id": "win_ts_firewall",
        "document": "Брандмауэр Windows:\n1. wf.msc — Windows Firewall с расширенными настройками\n2. Get-NetFirewallRule | Where Enabled -eq True — активные правила\n3. New-NetFirewallRule -DisplayName Allow_SSH -Direction Inbound -Port 22 -Protocol TCP -Action Allow\n4. Remove-NetFirewallRule -DisplayName Allow_SSH\n5. Set-NetFirewallRule -DisplayName Rule -Enabled False — отключить правило\n6. netsh advfirewall set allprofiles state off — отключить файрвол\n7. netsh advfirewall set allprofiles state on — включить\n8. netsh advfirewall firewall show rule name=all — все правила\n9. Get-NetFirewallProfile — статус профилей (Domain/Private/Public)\n10. netsh advfirewall export backup.wfw — бэкап правил",
        "metadata": {"tags": "firewall,NetFirewallRule,netsh,advfirewall,port,rule", "category": "windows_troubleshooting", "os": "windows"}
    },
    {
        "id": "win_ts_performance",
        "document": "Производительность Windows:\n1. perfmon — Performance Monitor, счётчики и графики\n2. resmon — Resource Monitor (CPU, Memory, Disk, Network)\n3. winsat formal — оценка производительности системы\n4. Get-Counter \\Processor(_Total)\\% Processor Time — загрузка CPU\n5. Get-Counter \\Memory\\Available MBytes — свободная RAM\n6. Optimize-Volume -DriveLetter C -ReTrim — TRIM для SSD\n7. powercfg /energy — отчёт энергопотребления\n8. powercfg /batteryreport — отчёт батареи (ноутбук)\n9. SystemPropertiesPerformance.exe — визуальные эффекты\n10. bcdedit /set disabledynamictick yes — точность таймера",
        "metadata": {"tags": "performance,perfmon,resmon,optimization,SSD,CPU,RAM", "category": "windows_troubleshooting", "os": "windows"}
    },
    {
        "id": "win_ts_recovery",
        "document": "Восстановление системы Windows:\n1. rstrui.exe — System Restore GUI\n2. Checkpoint-Computer -Description Before_changes — создать точку\n3. Get-ComputerRestorePoint — список точек восстановления\n4. Restore-Computer -RestorePoint N — восстановление\n5. Safe Mode: Settings > Recovery > Advanced startup > Restart; или msconfig > Boot > Safe boot\n6. WinRE: Shift+Restart или 3x принудительная перезагрузка\n7. bcdedit /set {default} safeboot minimal — Safe Mode через bcdedit\n8. reagentc /info — статус Windows Recovery Environment\n9. Сброс: Settings > System > Recovery > Reset this PC\n10. DISM /Online /Cleanup-Image /RestoreHealth — ремонт образа",
        "metadata": {"tags": "recovery,restore,SafeMode,WinRE,checkpoint,reset,bcdedit", "category": "windows_troubleshooting", "os": "windows"}
    },
    {
        "id": "win_ts_bluetooth",
        "document": "Bluetooth Windows:\n1. Settings > Bluetooth & devices — включение/выключение\n2. Get-PnpDevice -Class Bluetooth — список BT устройств\n3. devmgmt.msc > Bluetooth — Device Manager\n4. Службы: Get-Service bthserv, BTAGService — должны быть Running\n5. Restart-Service bthserv -Force — перезапуск стека\n6. Удаление устройства: Settings > Bluetooth > устройство > Remove\n7. fsquirt — Bluetooth File Transfer wizard\n8. Сброс: удалить устройство в Device Manager > Scan for hardware changes\n9. Troubleshooter: msdt.exe /id BluetoothDiagnostic\n10. Логи: Get-WinEvent -LogName Microsoft-Windows-Bluetooth* | Select -First 20",
        "metadata": {"tags": "bluetooth,BT,pairing,bthserv,wireless,device", "category": "windows_troubleshooting", "os": "windows"}
    },
    {
        "id": "win_ts_printers",
        "document": "Принтеры Windows:\n1. Get-Printer — список установленных принтеров\n2. Get-PrintJob -PrinterName Name — очередь печати\n3. Remove-PrintJob — удаление заданий из очереди\n4. Restart-Service Spooler -Force — перезапуск службы печати\n5. Очистка очереди: Stop-Service Spooler; Remove-Item C:\\Windows\\System32\\spool\\PRINTERS\\*; Start-Service Spooler\n6. Add-Printer -Name MyPrinter -DriverName Driver -PortName IP_port\n7. printmanagement.msc — GUI управления принтерами\n8. Get-PrinterDriver — список драйверов принтеров\n9. rundll32 printui.dll,PrintUIEntry /il — установка принтера через wizard\n10. Troubleshooter: msdt.exe /id PrinterDiagnostic",
        "metadata": {"tags": "printer,spooler,print,queue,driver,printmanagement", "category": "windows_troubleshooting", "os": "windows"}
    },
    {
        "id": "win_ts_powershell_basics",
        "document": "PowerShell basics:\n1. Get-Command *keyword* — поиск команд\n2. Get-Help CommandName -Full — полная справка\n3. Get-Help CommandName -Examples — примеры использования\n4. Pipeline: Get-Process | Where CPU -gt 10 | Sort CPU -Desc | Select -First 5\n5. Get-ExecutionPolicy; Set-ExecutionPolicy RemoteSigned -Scope CurrentUser\n6. Профиль: $PROFILE — путь к файлу профиля, notepad $PROFILE\n7. Алиасы: Get-Alias, Set-Alias ll Get-ChildItem\n8. Переменные: $var = value; $env:PATH; [Environment]::GetEnvironmentVariable()\n9. Скрипты: .\\script.ps1, & path_with_spaces\\script.ps1\n10. Модули: Get-Module -ListAvailable, Install-Module Name -Scope CurrentUser",
        "metadata": {"tags": "powershell,basics,Get-Command,Get-Help,pipeline,execution-policy", "category": "windows_troubleshooting", "os": "windows"}
    },
]

ts.upsert(
    ids=[s["id"] for s in ts_skills],
    documents=[s["document"] for s in ts_skills],
    metadatas=[s["metadata"] for s in ts_skills],
)
print(f"windows_troubleshooting: {ts.count()} skills")

# === Collection 2: windows_powershell ===
ps = client.get_or_create_collection("windows_powershell")

ps_skills = [
    {
        "id": "ps_filesystem",
        "document": "PowerShell файловая система:\n1. Get-ChildItem (ls/dir) — содержимое директории; -Recurse -Filter *.txt\n2. Get-Item path — информация о файле/папке\n3. Set-Item, Set-Content, Add-Content — запись в файл\n4. Copy-Item src dst -Recurse — копирование\n5. Move-Item src dst — перемещение/переименование\n6. Remove-Item path -Recurse -Force — удаление\n7. New-Item -ItemType Directory -Path folder — создание папки\n8. New-Item -ItemType File -Path file.txt -Value content\n9. Test-Path path — проверка существования\n10. Resolve-Path *.log — разрешение wildcard путей\n11. Get-Content file.txt -Tail 20 — последние 20 строк (tail)",
        "metadata": {"tags": "filesystem,Get-ChildItem,Copy-Item,Move-Item,Remove-Item,Test-Path", "category": "powershell", "os": "windows"}
    },
    {
        "id": "ps_processes_services",
        "document": "PowerShell процессы и службы:\n1. Get-Process — все процессы; Get-Process -Name chrome\n2. Stop-Process -Name notepad -Force; Stop-Process -Id 1234\n3. Start-Process notepad; Start-Process cmd -Verb RunAs (админ)\n4. Wait-Process -Name setup — ждать завершения\n5. Get-Service — все службы; Get-Service -Name wuauserv\n6. Start-Service, Stop-Service, Restart-Service -Name ServiceName\n7. Set-Service -Name ssh-agent -StartupType Automatic\n8. Get-Process | Measure-Object WorkingSet -Sum — общая RAM\n9. Get-Service | Where Status -eq Running | Measure-Object — кол-во запущенных",
        "metadata": {"tags": "process,service,Get-Process,Get-Service,Stop-Process,Start-Service", "category": "powershell", "os": "windows"}
    },
    {
        "id": "ps_network",
        "document": "PowerShell сеть:\n1. Test-NetConnection google.com — пинг + трассировка\n2. Test-NetConnection -ComputerName server -Port 22 — проверка порта\n3. Resolve-DnsName domain.com — DNS запрос\n4. Get-NetAdapter — список сетевых адаптеров\n5. Get-NetIPAddress — IP адреса на интерфейсах\n6. Get-NetRoute — таблица маршрутизации\n7. Get-DnsClientServerAddress — DNS серверы\n8. Set-DnsClientServerAddress -InterfaceAlias Ethernet -ServerAddresses 8.8.8.8,8.8.4.4\n9. Get-NetTCPConnection — активные TCP соединения (netstat)\n10. Test-Connection -ComputerName 8.8.8.8 -Count 4 — аналог ping",
        "metadata": {"tags": "network,Test-NetConnection,DNS,NetAdapter,NetIPAddress,TCP", "category": "powershell", "os": "windows"}
    },
    {
        "id": "ps_registry",
        "document": "PowerShell реестр:\n1. Get-ItemProperty -Path HKLM:\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\n2. Set-ItemProperty -Path HKCU:\\path -Name Key -Value Data\n3. New-ItemProperty -Path HKCU:\\path -Name Key -Value Data -PropertyType String\n4. Remove-ItemProperty -Path HKCU:\\path -Name Key\n5. New-Item -Path HKCU:\\Software\\NewKey — создать ключ\n6. Remove-Item -Path HKCU:\\Software\\NewKey -Recurse\n7. Test-Path HKLM:\\SOFTWARE\\path — проверка существования\n8. Get-ChildItem HKLM:\\SOFTWARE\\Microsoft — обзор подключей\n9. PSDrive: HKLM:, HKCU: — навигация как файловая система\n10. PropertyType: String, DWord, QWord, Binary, ExpandString, MultiString",
        "metadata": {"tags": "registry,HKLM,HKCU,Get-ItemProperty,Set-ItemProperty,New-ItemProperty", "category": "powershell", "os": "windows"}
    },
    {
        "id": "ps_users",
        "document": "PowerShell пользователи:\n1. Get-LocalUser — список локальных пользователей\n2. New-LocalUser -Name user -Password (ConvertTo-SecureString pass -AsPlainText -Force)\n3. Remove-LocalUser -Name user\n4. Enable-LocalUser / Disable-LocalUser -Name user\n5. Get-LocalGroup — список групп\n6. Add-LocalGroupMember -Group Administrators -Member user\n7. Remove-LocalGroupMember -Group Administrators -Member user\n8. Get-LocalGroupMember -Group Administrators — члены группы\n9. whoami /all — текущий пользователь, группы, привилегии\n10. [System.Security.Principal.WindowsIdentity]::GetCurrent().Name",
        "metadata": {"tags": "user,LocalUser,LocalGroup,administrator,whoami,account", "category": "powershell", "os": "windows"}
    },
    {
        "id": "ps_wmi_cim",
        "document": "PowerShell WMI/CIM:\n1. Get-CimInstance Win32_OperatingSystem — инфо об ОС, uptime, RAM\n2. Get-CimInstance Win32_ComputerSystem — имя, домен, RAM, процессор\n3. Get-CimInstance Win32_Processor — CPU: имя, ядра, частота\n4. Get-CimInstance Win32_DiskDrive — физические диски\n5. Get-CimInstance Win32_LogicalDisk — логические диски, свободное место\n6. Get-CimInstance Win32_NetworkAdapter | Where NetEnabled — сетевые адаптеры\n7. Get-CimInstance Win32_BIOS — версия BIOS\n8. Get-CimInstance Win32_VideoController — GPU info\n9. Get-CimInstance Win32_Product — установленные программы (медленно!)\n10. Invoke-CimMethod -ClassName Win32_Process -MethodName Create -Arguments @{CommandLine=notepad}",
        "metadata": {"tags": "WMI,CIM,Win32_OperatingSystem,Win32_DiskDrive,hardware,system-info", "category": "powershell", "os": "windows"}
    },
    {
        "id": "ps_eventlog",
        "document": "PowerShell Event Log:\n1. Get-EventLog -LogName System -Newest 20 — последние системные события\n2. Get-EventLog -LogName Application -EntryType Error -Newest 10\n3. Get-WinEvent -LogName System -MaxEvents 20 — новый API\n4. Get-WinEvent -FilterHashtable @{LogName=System; Level=2; StartTime=(Get-Date).AddDays(-1)}\n5. Level: 1=Critical, 2=Error, 3=Warning, 4=Information\n6. Get-WinEvent -ListLog * | Where RecordCount -gt 0 — логи с записями\n7. Get-EventLog -LogName Security -InstanceId 4625 — неудачные входы\n8. wevtutil qe System /c:10 /f:text — через wevtutil\n9. Clear-EventLog -LogName Application — очистка лога\n10. Get-WinEvent -LogName Microsoft-Windows-* — специализированные логи",
        "metadata": {"tags": "eventlog,Get-EventLog,Get-WinEvent,log,error,security,audit", "category": "powershell", "os": "windows"}
    },
    {
        "id": "ps_packages",
        "document": "PowerShell packages:\n1. winget search name — поиск пакета\n2. winget install Publisher.App — установка\n3. winget upgrade --all — обновление всех пакетов\n4. winget list — установленные пакеты\n5. winget uninstall App — удаление\n6. winget export -o packages.json; winget import -i packages.json\n7. choco install packagename -y — Chocolatey\n8. choco upgrade all -y — обновить всё через Chocolatey\n9. scoop install app — Scoop (в userspace)\n10. Install-Module ModuleName -Scope CurrentUser — PS Gallery\n11. Find-Module *keyword* — поиск в PS Gallery",
        "metadata": {"tags": "package,winget,chocolatey,scoop,install,upgrade,module", "category": "powershell", "os": "windows"}
    },
    {
        "id": "ps_remote",
        "document": "PowerShell remote:\n1. Enable-PSRemoting -Force — включить WinRM на целевой машине\n2. Enter-PSSession -ComputerName server — интерактивная сессия\n3. Invoke-Command -ComputerName server -ScriptBlock { Get-Process }\n4. Invoke-Command -ComputerName s1,s2,s3 -ScriptBlock { ... } — параллельно\n5. New-PSSession -ComputerName server — создать persistent сессию\n6. $s = New-PSSession server; Invoke-Command -Session $s -ScriptBlock { ... }\n7. Copy-Item -Path local -Destination remote -ToSession $s\n8. winrm quickconfig — быстрая настройка WinRM\n9. Set-Item WSMan:\\localhost\\Client\\TrustedHosts -Value server\n10. Test-WSMan -ComputerName server — проверка WinRM",
        "metadata": {"tags": "remote,PSSession,WinRM,Invoke-Command,remoting,SSH", "category": "powershell", "os": "windows"}
    },
    {
        "id": "ps_scheduled_tasks",
        "document": "PowerShell scheduled tasks:\n1. Get-ScheduledTask — список всех задач\n2. Get-ScheduledTask | Where State -eq Ready — активные задачи\n3. $trigger = New-ScheduledTaskTrigger -Daily -At 9am\n4. $action = New-ScheduledTaskAction -Execute pwsh.exe -Argument -File script.ps1\n5. Register-ScheduledTask -TaskName MyTask -Trigger $trigger -Action $action -User SYSTEM\n6. Unregister-ScheduledTask -TaskName MyTask -Confirm:$false\n7. Start-ScheduledTask -TaskName MyTask — запуск вручную\n8. Set-ScheduledTask — изменение существующей задачи\n9. Disable-ScheduledTask / Enable-ScheduledTask -TaskName MyTask\n10. Get-ScheduledTaskInfo -TaskName MyTask — последний запуск, результат",
        "metadata": {"tags": "ScheduledTask,trigger,action,cron,automation,timer", "category": "powershell", "os": "windows"}
    },
    {
        "id": "ps_environment",
        "document": "PowerShell environment:\n1. $env:PATH — текущий PATH\n2. $env:USERPROFILE, $env:APPDATA, $env:TEMP — стандартные переменные\n3. Get-ChildItem env: — все переменные окружения\n4. $env:MY_VAR = value — установка для текущей сессии\n5. [Environment]::SetEnvironmentVariable(VAR, value, User) — постоянно для юзера\n6. [Environment]::SetEnvironmentVariable(VAR, value, Machine) — для системы (админ)\n7. [Environment]::GetEnvironmentVariable(PATH, User) — чтение user PATH\n8. Добавление в PATH: получить старый, добавить ;C:\\new, сохранить\n9. Удаление: [Environment]::SetEnvironmentVariable(VAR, $null, User)\n10. refreshenv (Chocolatey) — обновить переменные без перезапуска",
        "metadata": {"tags": "environment,PATH,env,variable,SetEnvironmentVariable,system", "category": "powershell", "os": "windows"}
    },
    {
        "id": "ps_disk_management",
        "document": "PowerShell disk management:\n1. Get-Disk — список физических дисков\n2. Get-Partition — список разделов\n3. Get-Volume — список томов с буквами и размерами\n4. Initialize-Disk -Number 1 -PartitionStyle GPT\n5. New-Partition -DiskNumber 1 -UseMaximumSize -AssignDriveLetter\n6. Format-Volume -DriveLetter E -FileSystem NTFS -NewFileSystemLabel Data\n7. Optimize-Volume -DriveLetter C -Defrag — дефрагментация\n8. Optimize-Volume -DriveLetter C -ReTrim — TRIM для SSD\n9. Resize-Partition -DriveLetter C -Size 100GB\n10. Get-PhysicalDisk — тип (SSD/HDD), здоровье, размер\n11. Set-Disk -Number 1 -IsOffline $false — включить диск",
        "metadata": {"tags": "disk,Get-Volume,Get-Disk,Get-Partition,format,SSD,NTFS", "category": "powershell", "os": "windows"}
    },
    {
        "id": "ps_windows_update",
        "document": "PowerShell Windows Update:\n1. Install-Module PSWindowsUpdate -Scope CurrentUser -Force\n2. Import-Module PSWindowsUpdate\n3. Get-WindowsUpdate — список доступных обновлений\n4. Install-WindowsUpdate -AcceptAll — установить все\n5. Install-WindowsUpdate -AcceptAll -AutoReboot — с автоперезагрузкой\n6. Get-WUHistory — история обновлений\n7. Hide-WindowsUpdate -KBArticleID KB1234567 — скрыть обновление\n8. Show-WindowsUpdate -KBArticleID KB1234567 — показать скрытое\n9. Get-WUInstallerStatus — статус установщика\n10. Remove-WindowsUpdate -KBArticleID KB1234567 — удалить обновление",
        "metadata": {"tags": "WindowsUpdate,PSWindowsUpdate,KB,patch,update,install", "category": "powershell", "os": "windows"}
    },
    {
        "id": "ps_security",
        "document": "PowerShell security:\n1. Get-ExecutionPolicy -List — политики по скоупам\n2. Set-ExecutionPolicy RemoteSigned -Scope CurrentUser\n3. Скоупы: MachinePolicy, UserPolicy, Process, CurrentUser, LocalMachine\n4. Политики: Restricted, AllSigned, RemoteSigned, Unrestricted, Bypass\n5. Unblock-File -Path script.ps1 — разблокировка скачанного файла\n6. Get-AuthenticodeSignature file.ps1 — проверка подписи\n7. Set-AuthenticodeSignature — подпись скрипта сертификатом\n8. ConvertTo-SecureString pass -AsPlainText -Force — безопасная строка\n9. Get-Credential — запрос логина/пароля\n10. [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.SecurityProtocolType]::Tls12",
        "metadata": {"tags": "security,ExecutionPolicy,signature,credential,TLS,Unblock-File", "category": "powershell", "os": "windows"}
    },
    {
        "id": "ps_output",
        "document": "PowerShell output:\n1. Format-Table -AutoSize — табличный вывод\n2. Format-List -Property * — все свойства списком\n3. Format-Wide -Column 3 — широкий формат\n4. Export-Csv -Path data.csv -NoTypeInformation — экспорт в CSV\n5. Import-Csv -Path data.csv — импорт CSV\n6. ConvertTo-Json -Depth 5 — конвертация в JSON\n7. ConvertFrom-Json — парсинг JSON\n8. Out-File -FilePath output.txt — запись в файл\n9. Tee-Object -FilePath log.txt — и на экран, и в файл\n10. Select-Object Name,CPU,WorkingSet — выбор свойств\n11. Sort-Object CPU -Descending — сортировка\n12. Group-Object Status — группировка\n13. Measure-Object -Property Size -Sum -Average — статистика",
        "metadata": {"tags": "output,Format-Table,Export-Csv,ConvertTo-Json,Sort-Object,Select-Object", "category": "powershell", "os": "windows"}
    },
]

ps.upsert(
    ids=[s["id"] for s in ps_skills],
    documents=[s["document"] for s in ps_skills],
    metadatas=[s["metadata"] for s in ps_skills],
)
print(f"windows_powershell: {ps.count()} skills")

# Summary
total = ts.count() + ps.count()
print(f"\nTotal: {total} skills added to MemPalace")
cols = client.list_collections()
print(f"Collections: {[c.name for c in cols]}")
