
# Landmark API

A RESTful API built with Go that provides information about famous landmarks around the world, using GORM as the ORM.

## Project Structure
```
landmark-api/
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── api/
│   │   ├── handlers/
│   │   │   └── landmarks.go
│   │   ├── middleware/
│   │   │   └── api_key.go
│   │   └── routes.go
│   ├── db/
│   │   └── database.go
│   ├── models/
│   │   ├── landmark.go
│   │   └── subscription.go
│   └── services/
│       ├── api_key.go
│       ├── auth.go
│       └── landmark.go
├── .env
├── go.mod
├── go.sum
└── README.md
```

## Prerequisites

- Go 1.19 or later
- PostgreSQL 12 or later

## Setup Instructions

1. Clone the repository:
```bash
git clone <repository-url>
cd landmark-api
```

2. Initialize the Go module:
```bash
go mod init landmark-api
```

3. Install dependencies:
```bash
go get -u github.com/gorilla/mux
go get -u gorm.io/gorm
go get -u gorm.io/driver/postgres
go get -u github.com/joho/godotenv
```

4. Create a `.env` file:
```
DB_HOST=localhost
DB_PORT=5432
DB_USER=your_username
DB_PASSWORD=your_password
DB_NAME=landmark_db
JWT_SECRET=your_jwt_secret
```

## Running the Application

```bash
go run cmd/api/main.go
```

The server will start on port `8080`.

## API Endpoints

| Method | Endpoint                          | Description                                   |
|--------|-----------------------------------|-----------------------------------------------|
| GET    | /api/v1/landmarks                 | Get all landmarks                             |
| GET    | /api/v1/landmarks/{id}            | Get a specific landmark                       |
| GET    | /api/v1/landmarks/{id}/details    | Get detailed information for a landmark      |
| GET    | /api/v1/landmarks/country/{country} | Get landmarks by country                     |
| POST   | /api/v1/landmarks                 | Create a new landmark                         |
| PUT    | /api/v1/landmarks/{id}            | Update a landmark                             |
| DELETE | /api/v1/landmarks/{id}            | Delete a landmark                             |
| POST   | /auth/register                     | Register a new user                           |
| POST   | /auth/login                        | Login a user and return JWT token            |

## Authentication

The API uses JWT for authentication. To access protected routes, users need to register and login to obtain a token.

## API Keys

Each user has an associated API key, which must be included in the request headers as `x-api-key`. This allows users to manage their subscription plans and access different levels of detail based on their subscription tier.

## Example Requests/Responses

### Get All Landmarks
```
GET /api/v1/landmarks

Response:
[
  {
    "id": 1,
    "name": "Eiffel Tower",
    "country": "France",
    "city": "Paris",
    "description": "Iconic iron lattice tower on the Champ de Mars",
    "height": 324,
    "yearBuilt": 1889,
    "architect": "Gustave Eiffel",
    "visitorsPerYear": 7000000,
    "imageUrl": "/api/placeholder/800/600",
    "latitude": 48.8584,
    "longitude": 2.2945,
    "createdAt": "2024-02-20T10:00:00Z",
    "updatedAt": "2024-02-20T10:00:00Z"
  }
]
```

### Create a Landmark
```
POST /api/v1/landmarks
Content-Type: application/json

{
  "name": "Colosseum",
  "country": "Italy",
  "city": "Rome",
  "description": "Ancient amphitheater in the center of Rome",
  "height": 48,
  "yearBuilt": 80,
  "architect": "Vespasian",
  "visitorsPerYear": 7000000,
  "imageUrl": "/api/placeholder/800/600",
  "latitude": 41.8902,
  "longitude": 12.4922
}
```

## Development

### Running Tests
```bash
go test ./...
```

## License
This project is licensed under the MIT License - see the LICENSE file for details.
