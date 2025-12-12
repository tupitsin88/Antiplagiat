# Antiplagiat — Система проверки работ на плагиат

Микросервисная система для проверки студенческих работ на плагиат. Состоит из двух сервисов (хранение и анализ), управляется через API Gateway.

## Запуск

```bash
cd docker
docker compose up --build -d
```

Swagger UI на `http://localhost:8083`

## Архитектура

### Компоненты системы

**File Storing Service (порт 8081)**
- Хранение файлов в MinIO (S3)
- Метаданные работ в PostgreSQL (таблица `works`)
- Валидация расширения файлов (только .txt)

**File Analysis Service (порт 8082)**
- Анализ текстов на плагиат
- Сохранение отчетов в PostgreSQL (таблица `plagiat_reports`)
- Генерация облака слов через QuickChart API

**API Gateway (Nginx, порт 8080)**
- Единая точка входа
- Маршрутизация запросов к сервисам
- Swagger UI

**Инфраструктура**
- PostgreSQL 15 (единая БД для обоих сервисов)
- MinIO (S3-совместимое хранилище)

## API

### File Storing Service

- `POST /upload` - загрузить работу (multipart/form-data)
  - student_name, assignment_name, file (.txt)
  - Возвращает: work_id
  
- `GET /get` - список всех работ
  
- `GET /get/{id}` - информация о работе
  
- `GET /download/{id}` - скачать файл

### File Analysis Service

- `POST /check/{id}` - проверить работу на плагиат
  - Возвращает: report_id
  
- `GET /get-report/{id}` - получить отчет
  - Возвращает: plagiat_score (%), plagiat_sources, word_cloud_url

## Сценарии взаимодействия

### Сценарий 1: Загрузка работы

```
Клиент → API Gateway → File Storing Service

1. POST /api/store/upload
   - Nginx маршрутизирует на /upload в File Storing
   - Валидация расширения файла (.txt)
   - INSERT в таблицу works (student_name, assignment_name, uploaded_at)
   - Получаемое work_id используется как имя файла в MinIO
   - PutObject в MinIO: objectName = "{work_id}.txt"
   
Ответ: { work_id, status: "uploaded", file: "42.txt" }
```

### Сценарий 2: Проверка на плагиат

```
1. Клиент → API Gateway → File Storing Service
   GET /api/store/get/{work_id}
   - Nginx маршрутизирует на /get/{work_id} в File Storing
   - SELECT * FROM works WHERE id = ?
   
Ответ: { id, student_name, assignment_name, uploaded_at }

2. Преподаватель инициирует проверку:
   Клиент → API Gateway → File Analysis Service
   POST /api/analysis/check/{work_id}
   - Nginx маршрутизирует на /check/{work_id} в File Analysis

3. File Analysis Service начинает анализ:
   
   3.1 Скачивание исходного файла:
       HTTP GET http://file-storing-service:8081/download/{work_id}
       - Если ошибка → retry 3 раза с exponential backoff
       - Получает содержимое файла (text)
   
   3.2 Получение метаданных работы:
       HTTP GET http://file-storing-service:8081/get/{work_id}
       - Извлекает assignment_name
       - Используется для поиска конкурирующих работ
   
   3.3 Поиск конкурирующих работ в БД:
       SELECT id, student_name FROM works 
       WHERE assignment_name = ? AND id != ?
       - Находит все работы по тому же заданию
   
   3.4 Сравнение с каждой конкурирующей работой:
       for each work_id in competitors:
           HTTP GET http://file-storing-service:8081/download/{competitor_id}
           - Если ошибка → логируется, работа пропускается (graceful degradation)
           - calculateSimilarity(text, competitor_text)
           - Работа добавляется в список обнаруженных источников плагиата, и если ее процент выше текущего максимума, он становится итоговой оценкой плагиата
   
   3.5 Генерация облака слов:
       URL = https://quickchart.io/wordcloud?text=...
       - Первые 3000 символов текста
   
   3.6 Сохранение отчета в БД:
       INSERT INTO plagiat_reports 
       (work_id, plagiat_score, plagiat_sources, word_cloud_url, checked_at)
       VALUES (?, ?, ?, ?, now())
       
Ответ: { status: "success", id: report_id, work_id }
```

### Сценарий 3: Получение отчета

```
Клиент → API Gateway → File Analysis Service → PostgreSQL

1. GET /api/analysis/get-report/{report_id}
   - Nginx маршрутизирует на /get-report/{report_id}
   - SELECT * FROM plagiat_reports WHERE id = ?
   
Ответ: { id, work_id, plagiat_score, plagiat_sources, word_cloud_url, checked_at }
```

## Обработка ошибок

**Retry логика (File Analysis Service)**
- При скачивании файла из File Storing: 3 попытки с задержкой 200ms, 400ms, 600ms
- Если File Storing недоступен - анализ падает с ошибкой

**Graceful Degradation (File Analysis Service)**
- Если одна из конкурирующих работ не скачивается - пропускается
- Анализ продолжается со следующей работой
- В отчете добавляется пометка: "(skipped N works due to errors)"
- Результат получается неполный, но система не падает

**Graceful Shutdown**
- Оба сервиса ждут 5 секунд на завершение текущих запросов
- После этого прерывают долгие операции

## Алгоритм сравнения (Jaccard Index)

```
text1 = "hello world test"
text2 = "hello world python"

words1 = {hello, world, test}
words2 = {hello, world, python}

intersection = {hello, world} (2 слова)
score = intersection / len(words1) * 100%
     = 2 / 3 * 100%
     = 66.7%
```

Если score > 50% → источник плагиата найден

## Технологии

- Go 1.20
- PostgreSQL 15
- MinIO (S3)
- Nginx
- Docker Compose
- Swagger (OpenAPI 3.0)