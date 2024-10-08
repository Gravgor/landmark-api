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
│   │   └── routes.go
│   ├── db/
│   │   └── database.go
│   └── models/
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

4. Create a .env file:
```
DB_HOST=localhost
DB_PORT=5432
DB_USER=your_username
DB_PASSWORD=your_password
DB_NAME=landmark_db
```

## Running the Application

```bash
go run cmd/api/main.go
```

The server will start on port 8080.

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET    | /api/landmarks | Get all landmarks |
| GET    | /api/landmarks/{id} | Get a specific landmark |
| GET    | /api/landmarks/country/{country} | Get landmarks by country |
| POST   | /api/landmarks | Create a new landmark |
| PUT    | /api/landmarks/{id} | Update a landmark |
| DELETE | /api/landmarks/{id} | Delete a landmark |

## Example Requests/Responses

### Get All Landmarks
```
GET /api/landmarks

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
POST /api/landmarks
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
This project is licensed under the MIT License - see the LICENSE file for details