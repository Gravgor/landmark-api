# Landmark API

A robust RESTful API service for managing and retrieving information about landmarks worldwide. The API provides detailed information about historical sites, monuments, and points of interest, with different access levels based on subscription tiers.

## 🌟 Features

- **Authentication & Authorization**
  - JWT-based authentication
  - Role-based access control
  - API key management
  - Subscription-based access tiers (Free, Pro, Enterprise)

- **Landmark Information**
  - Comprehensive landmark details
  - Geolocation data
  - Historical information
  - Visitor information
  - Accessibility details
  - Opening hours and ticket prices
  - Live Data (Weather, Public transport)

- **Advanced Querying**
  - Field selection
  - Pagination
  - Sorting
  - Filtering
  - Full-text search
  - Search by coordinates with radius
  - Search by category

- **Performance & Scalability**
  - Redis caching
  - Rate limiting
  - Connection pooling
  - Optimized database queries

## 🚀 Getting Started

### Prerequisites

- Go 1.19 or higher
- PostgreSQL 13 or higher
- Redis 6.x or higher
- Docker (optional)

### Environment Setup

1. Clone the repository:
```bash
git clone https://github.com/Gravgor/landmark-api.git
cd landmark-api
```

2. Create a `.env` file in the project root:
```env
# Server Configuration
PORT=5050
ENV=development

# Database Configuration
DATABASE_URL=postgresql://username:password@localhost:5432/landmark_db?sslmode=disable

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=your_redis_password

# JWT Configuration
JWT_SECRET=your_jwt_secret_key

# Rate Limiting
RATE_LIMIT=100
RATE_LIMIT_DURATION=1h
```

### Running the Application

#### Local Development

1. Install dependencies:
```bash
go mod download
```

2. Start the database and Redis (if using Docker):
```bash
docker-compose up -d postgres redis
```

3. Run migrations:
```bash
go run cmd/migrate/main.go
```

4. Start the server:
```bash
go run main.go
```

#### Using Docker

```bash
docker-compose up -d
```

## 📖 API Documentation

### Authentication

#### Register a new user
```http
POST /auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepassword",
  "name": "John Doe"
}
```

#### Login
```http
POST /auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepassword"
}
```

### Landmarks

#### Get all landmarks
```http
GET /api/v1/landmarks
Authorization: Bearer <your_jwt_token>
X-API-Key: <your_api_key>
```

Query Parameters:
- `limit` (default: 10)
- `offset` (default: 0)
- `sort` (e.g., "-name" for descending order)
- `fields` (comma-separated list of fields)
- Additional filters as query parameters

#### Get landmark by ID
```http
GET /api/v1/landmarks/{id}
Authorization: Bearer <your_jwt_token>
X-API-Key: <your_api_key>
```

#### Get landmarks by country
```http
GET /api/v1/landmarks/country/{country}
Authorization: Bearer <your_jwt_token>
X-API-Key: <your_api_key>
```

#### Search landmarks by name
```http
GET /api/v1/landmarks/name/{name}
Authorization: Bearer <your_jwt_token>
X-API-Key: <your_api_key>
```

### Subscription Tiers

| Feature                    | Free Plan | Pro Plan | Enterprise Plan |
|---------------------------|-----------|-----------|-----------------|
| Basic landmark info       | ✓         | ✓         | ✓               |
| Detailed descriptions     | ✗         | ✓         | ✓               |
| Historical significance   | ✗         | ✓         | ✓               |
| Visitor tips              | ✗         | ✓         | ✓               |
| Real-time data           | ✗         | ✗         | ✓               |
| Rate limit               | 100/hour  | 1000/hour | Unlimited       |

## 🛠 Project Structure

```
landmark-api/
├── cmd/
│   └── migrate/
│       └── main.go
├── internal/
│   ├── api/
│   │   └── handlers/
│   ├── cache/
│   ├── middleware/
│   ├── models/
│   ├── repository/
│   └── services/
├── migrations/
├── docker-compose.yml
├── Dockerfile
├── go.mod
├── go.sum
├── main.go
└── README.md
```

## 🔒 Security

- All endpoints except `/auth/register` and `/auth/login` require authentication
- Passwords are hashed using bcrypt
- Rate limiting is implemented per API key
- Input validation and sanitization
- Prepared statements for database queries
- Environment-based configuration

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Gorilla Mux](https://github.com/gorilla/mux) for routing
- [GORM](https://gorm.io) for database operations
- [JWT-Go](https://github.com/golang-jwt/jwt) for JWT authentication
- [Go-Redis](https://github.com/go-redis/redis) for caching

## 📧 Contact

Marceli Borowczak - marceliborowczak@example.com

Project Link: [https://github.com/Gravgor/landmark-api](https://github.com/Gravgor/landmark-api)
