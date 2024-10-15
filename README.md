## Landmark API

This API provides information about landmarks, it's designed to be used by clients that need to access and manage landmark data. The API uses JWT-based authentication and API keys for authorization.

### Inputs

*   **Authentication:** User registration and login require providing email and password. Requires `Bearer` token.
*   **API Key:** Access to API endpoints requires providing a valid API key in the `x-api-key` header.
*   **Landmark Queries:** Various parameters can be used to filter and sort landmarks, such as `limit`, `offset`, `sort`, and `fields`.
*   **Search:** The API supports searching for landmarks by name, country, and proximity (latitude, longitude, and radius).
*   **File Upload:** Admin users can upload photos for landmarks using a multipart form request.
*   **Stripe Integration:** The API handles Stripe webhooks for subscription management, including checkout sessions and subscription updates.

### Outputs

*   **Landmark Data:** The API returns landmark data in JSON format, including basic information and detailed descriptions (based on subscription plan).
*   **Authentication Responses:** Responses for registration and login include a JWT token.
*   **Health Check:** The `/health` endpoint provides information about the API and database status.
*   **Usage Statistics:** Authenticated users can retrieve their API usage statistics.
*   **Request Logs:** Users can access logs of their API requests.
*   **Stripe Billing Information:** Users can get their Stripe billing information, including invoices and subscription details.
*   **File Upload Response:** The file upload endpoint returns the URL of the uploaded file on S3.
*   **Error Responses:** In case of errors, the API returns JSON responses with error details and appropriate status codes.