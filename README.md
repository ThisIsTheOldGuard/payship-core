payship-core/
├── cmd/
│   └── api/          # Точка входа приложения
│   └── worker/       # Точка входа воркера (будущий консьюмер)
├── internal/
│   ├── api/          # HTTP/gRPC хендлеры
│   ├── service/      # Бизнес-логика
│   └── repository/   # Работа с БД
├── pkg/              # Общий код (если будет)
├── configs/          # Конфиги
├── docker-compose.yml
├── Makefile
└── README.md

test changes