from io import IOBase
import requests
import json
from typing import List, Dict
import os
import time
import random
import logging
from requests_toolbelt.multipart.encoder import MultipartEncoder
import mimetypes
from contextlib import ExitStack

# Define the API endpoints and headers
UPLOAD_URL = "https://api.landmark-api.com/2079a66bb2f2859a721b9987ded608013fb38f95becb9d1e2b6520c5a06b8fd6/api/v1/landmarks/upload-photo"
CREATE_URL = "https://api.landmark-api.com/2079a66bb2f2859a721b9987ded608013fb38f95becb9d1e2b6520c5a06b8fd6/api/v1/landmarks/create"
UNSPLASH_URL = "https://api.unsplash.com/search/photos?client_id=CmOoJszifpwLyIhpB_QhjmMZ2Xsvc4SILzJv987G9oo"
HEADERS = {
    "x-api-key": "43f79790-bc83-47a5-ad99-ee965c27bc34",
    "Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MjkzNjY2MDUsInBsYW5fdHlwZSI6IlBSTyIsInJvbGUiOiJhZG1pbiIsInN1YnNjcmlwdGlvbl9pZCI6IjllYzRiYTcwLThkOTctNDY5OC05ZDllLWM2MTdkZGQyZjljNiIsInVzZXJfaWQiOiJkN2NlY2JhNS1iODFiLTRhMTItYWE3My0zZjcxYjNiZGI2NjMifQ.D9yc4Eqqflv0A2gpVTmp1-v0n1gz2NV04pBwNcF-ZLU"
}
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')

def find_images(landmark_name: str, num_images: int = 5) -> List[str]:
    """
    Find images for a given landmark using the Unsplash API.
    """
    params = {
        "query": landmark_name,
        "per_page": num_images,
    }
    response = requests.get(UNSPLASH_URL, params=params)
    if response.status_code == 200:
        data = response.json()
        return [photo["urls"]["regular"] for photo in data["results"]]
    else:
        print(f"Failed to fetch images for {landmark_name}: {response.text}")
        return []

def download_image(url: str, landmark_name: str, index: int) -> str:
    """
    Download an image from a URL and save it locally.
    """
    response = requests.get(url)
    if response.status_code == 200:
        filename = f"{landmark_name.replace(' ', '_')}_{index}.jpg"
        with open(filename, 'wb') as f:
            f.write(response.content)
        logging.info(f"Successfully downloaded image: {filename}")
        return os.path.abspath(filename)
    else:
        logging.error(f"Failed to download image from {url}")
        return ""


def upload_images(image_paths: List[str]) -> List[str]:
    """Upload images to the API and return the URLs of the uploaded images."""
    logging.info(f"Attempting to upload {len(image_paths)} images")
    
    uploaded_urls = []
    
    # Use a with statement to ensure files are properly closed
    with ExitStack() as stack:
        files = []
        for image_path in image_paths:
            if not os.path.exists(image_path):
                logging.error(f"File not found: {image_path}")
                continue
            
            try:
                file_size = os.path.getsize(image_path)
                logging.info(f"File {image_path} size: {file_size} bytes")
                
                # Open the file and add it to the ExitStack
                f = stack.enter_context(open(image_path, 'rb'))
                files.append(('images', (os.path.basename(image_path), f, 'image/jpeg')))
            except Exception as e:
                logging.error(f"Error processing {image_path}: {str(e)}")
        
        if not files:
            logging.error("No valid files to upload")
            return []
        
        try:
            logging.info(f"Sending POST request to {UPLOAD_URL} with {len(files)} files")
            response = requests.post(UPLOAD_URL, headers=HEADERS, files=files)
            logging.info(f"Response status code: {response.status_code}")
            logging.info(f"Response content: {response.text}")
            
            if response.status_code == 200:
                uploaded_urls = response.json().get('urls', [])
                logging.info(f"Successfully uploaded {len(uploaded_urls)} images")
            else:
                logging.error(f"Failed to upload images: {response.text}")
        except Exception as e:
            logging.error(f"Error during batch upload: {str(e)}")
    
    return uploaded_urls

def create_landmark(landmark: Dict, landmark_detail: Dict, image_urls: List[str]):
    """Create a new landmark entry in the API if image_urls is not empty."""
    if not image_urls:
        logging.warning(f"No images available for landmark: {landmark['name']}. Skipping creation.")
        return

    payload = {
        "landmark": landmark,
        "landmark_detail": landmark_detail,
        "image_urls": image_urls
    }
    
    logging.info(f"Sending POST request to create landmark: {landmark['name']}")
    logging.info(f"Payload: {json.dumps(payload, indent=2)}")
    
    response = requests.post(
        CREATE_URL,
        headers={**HEADERS, "Content-Type": "application/json"},
        data=json.dumps(payload)
    )
    
    if response.status_code == 201:
        logging.info(f"Successfully created landmark: {landmark['name']}")
    else:
        logging.error(f"Failed to create landmark {landmark['name']}: {response.text}")
        logging.error(f"Response status code: {response.status_code}")

def process_landmarks(landmarks: List[Dict]):
    """Process each landmark in the given list."""
    for landmark_data in landmarks:
        landmark = landmark_data['landmark']
        landmark_detail = landmark_data['landmark_detail']
        
        # Find images
        image_urls = find_images(landmark['name'])
        
        if not image_urls:
            logging.warning(f"No images found for {landmark['name']}. Skipping image upload.")
            create_landmark(landmark, landmark_detail, [])
            continue
        
        # Download images
        image_paths = []
        for i, url in enumerate(image_urls):
            path = download_image(url, landmark['name'], i)
            if path:
                image_paths.append(path)
        
        logging.info(f"Downloaded images: {image_paths}")
        
        # Upload images
        uploaded_urls = upload_images(image_paths)
        print(uploaded_urls)
        
        # Create landmark
        create_landmark(landmark, landmark_detail, uploaded_urls)
        
        # Clean up downloaded images
        for image_path in image_paths:
            try:
                os.remove(image_path)
                logging.info(f"Removed temporary file: {image_path}")
            except Exception as e:
                logging.error(f"Failed to remove temporary file {image_path}: {str(e)}")

# Example usage
if __name__ == "__main__":
    landmarks = [
              {
            "landmark": {
                "name": "Hollywood Sign",
                "description": "An iconic landmark and cultural symbol overlooking Hollywood, Los Angeles.",
                "latitude": 34.1341,
                "longitude": -118.3215,
                "country": "United States",
                "city": "Los Angeles",
                "category": "Cultural Landmark"
            },
            "landmark_detail": {
                "opening_hours": {
                    "Monday": "Open 24 hours",
                    "Tuesday": "Open 24 hours",
                    "Wednesday": "Open 24 hours",
                    "Thursday": "Open 24 hours",
                    "Friday": "Open 24 hours",
                    "Saturday": "Open 24 hours",
                    "Sunday": "Open 24 hours"
                },
                "ticket_prices": {
                    "Adult": "Free",
                    "Child": "Free"
                },
                "historical_significance": "Originally created in 1923 as an advertisement for a real estate development.",
                "visitor_tips": "Hike up to the sign for a closer view and great photo opportunities.",
                "accessibility_info": "Partially accessible"
            }
        },
        {
            "landmark": {
                "name": "Mount Vernon",
                "description": "The plantation home of George Washington, the first President of the United States.",
                "latitude": 38.7073,
                "longitude": -77.0861,
                "country": "United States",
                "city": "Mount Vernon",
                "category": "Historical Site"
            },
            "landmark_detail": {
                "opening_hours": {
                    "Monday": "09:00-17:00",
                    "Tuesday": "09:00-17:00",
                    "Wednesday": "09:00-17:00",
                    "Thursday": "09:00-17:00",
                    "Friday": "09:00-17:00",
                    "Saturday": "09:00-17:00",
                    "Sunday": "09:00-17:00"
                },
                "ticket_prices": {
                    "Adult": "28 USD",
                    "Child": "15 USD"
                },
                "historical_significance": "A National Historic Landmark, it was Washington's home and plantation.",
                "visitor_tips": "Take a guided tour to learn more about Washington's life and legacy.",
                "accessibility_info": "Fully accessible"
            }
        },
        {
            "landmark": {
                "name": "Golden Gate Park",
                "description": "A large urban park consisting of gardens, museums, and recreational areas.",
                "latitude": 37.7694,
                "longitude": -122.4862,
                "country": "United States",
                "city": "San Francisco",
                "category": "Park"
            },
            "landmark_detail": {
                "opening_hours": {
                    "Monday": "05:00-00:00",
                    "Tuesday": "05:00-00:00",
                    "Wednesday": "05:00-00:00",
                    "Thursday": "05:00-00:00",
                    "Friday": "05:00-00:00",
                    "Saturday": "05:00-00:00",
                    "Sunday": "05:00-00:00"
                },
                "ticket_prices": {
                    "Adult": "Free",
                    "Child": "Free"
                },
                "historical_significance": "Established in 1871, it is one of the most visited parks in the United States.",
                "visitor_tips": "Visit the Japanese Tea Garden and the de Young Museum.",
                "accessibility_info": "Fully accessible"
            }
        },
        {
            "landmark": {
                "name": "Independence Hall",
                "description": "The building where the Declaration of Independence and the United States Constitution were debated and adopted.",
                "latitude": 39.9489,
                "longitude": -75.1500,
                "country": "United States",
                "city": "Philadelphia",
                "category": "Historical Site"
            },
            "landmark_detail": {
                "opening_hours": {
                    "Monday": "09:00-17:00",
                    "Tuesday": "09:00-17:00",
                    "Wednesday": "09:00-17:00",
                    "Thursday": "09:00-17:00",
                    "Friday": "09:00-17:00",
                    "Saturday": "09:00-17:00",
                    "Sunday": "09:00-17:00"
                },
                "ticket_prices": {
                    "Adult": "Free (Timed entry tickets required)",
                    "Child": "Free"
                },
                "historical_significance": "A UNESCO World Heritage site, it is considered the birthplace of the United States.",
                "visitor_tips": "Reserve tickets in advance to ensure entry.",
                "accessibility_info": "Fully accessible"
            }
        }

    ]







    
    process_landmarks(landmarks)