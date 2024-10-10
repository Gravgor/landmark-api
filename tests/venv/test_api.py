import requests
import json

BASE_URL = "http://localhost:5050"  # Adjust based on your Go API URL

def log_response(response):
    """Utility function to log the status code and response JSON."""
    print(f"Status Code: {response.status_code}")
    try:
        response_json = response.json()
        print("Response JSON:")
        print(json.dumps(response_json, indent=4))
        return response_json
    except json.JSONDecodeError:
        print("No valid JSON in the response")
        return None

def handle_unexpected_status(response, expected_status):
    """Handle unexpected status codes and log useful info."""
    if response.status_code != expected_status:
        print(f"Error: Expected status {expected_status}, but got {response.status_code}")
        log_response(response)
        return True  # Indicate that an error was handled
    return False

def test_health_check():
    print("Testing Health Check...")
    response = requests.get(f"{BASE_URL}/health")
    
    if handle_unexpected_status(response, 200):
        return  # Exit if the status code is unexpected

    response_json = log_response(response)
    
    assert response_json["status"] == "API is running", "Expected status to be 'API is running'."
    assert "database" in response_json, "Expected 'database' key in the response."
    assert response_json["database"] == "Database connection is healthy", "Expected database connection to be healthy."
    assert "external_services" in response_json, "Expected 'external_services' key in the response."
    assert response_json["external_services"]["Weather API"] == "Available", "Expected 'Weather API' to be available."

def test_register():
    print("Testing User Registration...")
    url = f"{BASE_URL}/auth/register"
    payload = {"email": "testuser@example.com", "password": "password123"}
    
    response = requests.post(url, json=payload)
    
    if handle_unexpected_status(response, 201):
        return  # Exit if the status code is unexpected
    
    response_json = log_response(response)
    assert "token" in response_json, "Expected token to be present in the response."

def test_login():
    print("Testing User Login...")
    url = f"{BASE_URL}/auth/login"
    payload = {"email": "testuser@example.com", "password": "password123"}
    
    response = requests.post(url, json=payload)
    
    if handle_unexpected_status(response, 200):
        return  # Exit if the status code is unexpected

    response_json = log_response(response)
    assert "token" in response_json, "Expected token to be present in the response."

def test_get_landmarks():
    print("Testing Get Landmarks...")
    
    headers = {
        "Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3Mjg1OTg5ODgsInBsYW5fdHlwZSI6IlBSTyIsInN1YnNjcmlwdGlvbl9pZCI6IjM5NTk3NjJhLWJmMzYtNDBmZS1hNGViLTMzOThmODdmMzIyYyIsInVzZXJfaWQiOiJmZTkzMGFhZS03NmVkLTRlMWMtYmQwZS0xZjczOGE2MWYxMjQifQ.NZb0rHRk0MQ-VhHeNwMcevMBhYCs1Jhrcc8HwRhcJes", 
        "x-api-key": "58c1e7ad-a189-4424-a44b-4f77ddfc1714"  # Example API key, replace with actual
    }
    
    response = requests.get(f"{BASE_URL}/api/v1/landmarks", headers=headers)
    
    if handle_unexpected_status(response, 200):
        return  # Exit if the status code is unexpected
    
    landmarks = log_response(response)
    assert isinstance(landmarks, list), "Expected a list of landmarks."
    
    if landmarks:
        print("First landmark in the response:")
        print(json.dumps(landmarks[0], indent=4))  # Log the first landmark for inspection

def test_get_landmark_by_id():
    print("Testing Get Landmark by ID...")
    
    headers = {
        "Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3Mjg1OTg5ODgsInBsYW5fdHlwZSI6IlBSTyIsInN1YnNjcmlwdGlvbl9pZCI6IjM5NTk3NjJhLWJmMzYtNDBmZS1hNGViLTMzOThmODdmMzIyYyIsInVzZXJfaWQiOiJmZTkzMGFhZS03NmVkLTRlMWMtYmQwZS0xZjczOGE2MWYxMjQifQ.NZb0rHRk0MQ-VhHeNwMcevMBhYCs1Jhrcc8HwRhcJes", 
        "x-api-key": "58c1e7ad-a189-4424-a44b-4f77ddfc1714"
    }
    
    landmark_id = 1  # Replace with a valid landmark ID
    response = requests.get(f"{BASE_URL}/api/v1/landmarks/{landmark_id}", headers=headers)
    
    if handle_unexpected_status(response, 200):
        return  # Exit if the status code is unexpected
    
    landmark = log_response(response)
    assert isinstance(landmark, dict), "Expected a dictionary object for the landmark."

# Example of running all tests
if __name__ == "__main__":
    test_health_check()
    test_register()
    test_login()
    test_get_landmarks()
    test_get_landmark_by_id()
