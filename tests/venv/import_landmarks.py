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
    "Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3Mjk0NTI2MTIsInBsYW5fdHlwZSI6IlBSTyIsInJvbGUiOiJhZG1pbiIsInN1YnNjcmlwdGlvbl9pZCI6IjllYzRiYTcwLThkOTctNDY5OC05ZDllLWM2MTdkZGQyZjljNiIsInVzZXJfaWQiOiJkN2NlY2JhNS1iODFiLTRhMTItYWE3My0zZjcxYjNiZGI2NjMifQ.ZXqRSKl-E2Pc_tfV5vggy7Qco-Ios1UIiYlkCxlmhBw"
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
                "name": "The Valley of Kings",
                "description": "A famous archaeological site in Egypt, known for its numerous tombs of pharaohs and nobles from the New Kingdom.",
                "latitude": 25.7408,
                "longitude": 32.6010,
                "country": "Egypt",
                "city": "Luxor",
                "category": "Historical Landmark"
            },
            "landmark_detail": {
                "opening_hours": {
                    "Monday": "6:00 AM - 5:00 PM",
                    "Tuesday": "6:00 AM - 5:00 PM",
                    "Wednesday": "6:00 AM - 5:00 PM",
                    "Thursday": "6:00 AM - 5:00 PM",
                    "Friday": "6:00 AM - 5:00 PM",
                    "Saturday": "6:00 AM - 5:00 PM",
                    "Sunday": "6:00 AM - 5:00 PM"
                },
                "ticket_prices": {
                    "Adult": "$10",
                    "Child": "$5"
                },
                "historical_significance": "The burial site of many pharaohs, including Tutankhamun and Ramses II, dating back to the 16th to 11th century BC.",
                "visitor_tips": "Purchase a combined ticket to access multiple tombs and consider hiring a guide for deeper insights.",
                "accessibility_info": "Partially accessible"
            }
        }
    ]







    
    process_landmarks(landmarks)