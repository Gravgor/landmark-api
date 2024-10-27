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
UPLOAD_URL = "https://api.landmark-api.com/admin/landmarks/upload-photo"
CREATE_URL = "https://api.landmark-api.com/admin/landmarks/create"
UNSPLASH_URL = "https://api.unsplash.com/search/photos?client_id=CmOoJszifpwLyIhpB_QhjmMZ2Xsvc4SILzJv987G9oo"
HEADERS = {
    "x-api-key": "43f79790-bc83-47a5-ad99-ee965c27bc34",
    "Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MzAwMzUyNzIsInBsYW5fdHlwZSI6IlBSTyIsInJvbGUiOiJhZG1pbiIsInN1YnNjcmlwdGlvbl9pZCI6IjllYzRiYTcwLThkOTctNDY5OC05ZDllLWM2MTdkZGQyZjljNiIsInVzZXJfaWQiOiJkN2NlY2JhNS1iODFiLTRhMTItYWE3My0zZjcxYjNiZGI2NjMifQ.Bj6s-eGKwzElWX04HtUmk95fmBmoi4If1kHfm0K4L6w"
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

# Example United Statesge
if __name__ == "__main__":
    landmarks = []
    landmarks.extend([
        {
            "landmark": {
                "name": "Trinity College Dublin",
                "description": "Home to the famous Book of Kells and a historic university located in the heart of Dublin.",
                "latitude": 53.3454,
                "longitude": -6.2544,
                "country": "Ireland",
                "city": "Dublin",
                "category": "Educational Institution"
            },
            "landmark_detail": {
                "opening_hours": {
                    "Monday": "9:30 AM - 5:00 PM",
                    "Tuesday": "9:30 AM - 5:00 PM",
                    "Wednesday": "9:30 AM - 5:00 PM",
                    "Thursday": "9:30 AM - 5:00 PM",
                    "Friday": "9:30 AM - 5:00 PM",
                    "Saturday": "9:30 AM - 5:00 PM",
                    "Sunday": "9:30 AM - 5:00 PM"
                },
                "ticket_prices": {
                    "Adult": "€14",
                    "Child": "€5"
                },
                "historical_significance": "Founded in 1592, it is one of the oldest universities in the English-speaking world.",
                "visitor_tips": "Book tickets in advance to avoid long queues.",
                "accessibility_info": "Accessible facilities available."
            }
        },
        {
            "landmark": {
                "name": "Dublin Castle",
                "description": "A historic castle and government complex that has played a key role in Ireland's history.",
                "latitude": 53.3420,
                "longitude": -6.2675,
                "country": "Ireland",
                "city": "Dublin",
                "category": "Historical Building"
            },
            "landmark_detail": {
                "opening_hours": {
                    "Monday": "9:45 AM - 5:15 PM",
                    "Tuesday": "9:45 AM - 5:15 PM",
                    "Wednesday": "9:45 AM - 5:15 PM",
                    "Thursday": "9:45 AM - 5:15 PM",
                    "Friday": "9:45 AM - 5:15 PM",
                    "Saturday": "9:45 AM - 5:15 PM",
                    "Sunday": "9:45 AM - 5:15 PM"
                },
                "ticket_prices": {
                    "Adult": "€12",
                    "Child": "€6"
                },
                "historical_significance": "Built in the 13th century, it has served as a royal and governmental seat.",
                "visitor_tips": "Join a guided tour to learn more about its history.",
                "accessibility_info": "Fully accessible."
            }
        },
        {
            "landmark": {
                "name": "St. Patrick's Cathedral",
                "description": "The national cathedral of Ireland, known for its stunning architecture and historical significance.",
                "latitude": 53.3430,
                "longitude": -6.2701,
                "country": "Ireland",
                "city": "Dublin",
                "category": "Religious Building"
            },
            "landmark_detail": {
                "opening_hours": {
                    "Monday": "9:00 AM - 5:00 PM",
                    "Tuesday": "9:00 AM - 5:00 PM",
                    "Wednesday": "9:00 AM - 5:00 PM",
                    "Thursday": "9:00 AM - 5:00 PM",
                    "Friday": "9:00 AM - 5:00 PM",
                    "Saturday": "9:00 AM - 5:00 PM",
                    "Sunday": "9:00 AM - 5:00 PM"
                },
                "ticket_prices": {
                    "Adult": "€10",
                    "Child": "€5"
                },
                "historical_significance": "Founded in 1191, it is the largest cathedral in Ireland.",
                "visitor_tips": "Check for special services or concerts during your visit.",
                "accessibility_info": "Wheelchair accessible."
            }
        },
        {
            "landmark": {
                "name": "Christ Church Cathedral",
                "description": "A medieval cathedral known for its impressive architecture and vibrant history.",
                "latitude": 53.3439,
                "longitude": -6.2706,
                "country": "Ireland",
                "city": "Dublin",
                "category": "Religious Building"
            },
            "landmark_detail": {
                "opening_hours": {
                    "Monday": "9:30 AM - 5:30 PM",
                    "Tuesday": "9:30 AM - 5:30 PM",
                    "Wednesday": "9:30 AM - 5:30 PM",
                    "Thursday": "9:30 AM - 5:30 PM",
                    "Friday": "9:30 AM - 5:30 PM",
                    "Saturday": "9:30 AM - 5:30 PM",
                    "Sunday": "9:30 AM - 5:30 PM"
                },
                "ticket_prices": {
                    "Adult": "€8",
                    "Child": "€4"
                },
                "historical_significance": "Founded in 1030, it is one of Dublin's oldest buildings.",
                "visitor_tips": "Explore the crypt for a unique experience.",
                "accessibility_info": "Accessible facilities available."
            }
        },
        {
            "landmark": {
                "name": "Phoenix Park",
                "description": "One of the largest walled city parks in Europe, home to Dublin Zoo.",
                "latitude": 53.3602,
                "longitude": -6.3156,
                "country": "Ireland",
                "city": "Dublin",
                "category": "Park"
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
                "historical_significance": "Established in 1662, it spans over 1,750 acres.",
                "visitor_tips": "Visit the park for picnics and cycling.",
                "accessibility_info": "Accessible paths available."
            }
        },
        {
            "landmark": {
                "name": "The National Gallery of Ireland",
                "description": "Home to an extensive collection of European and Irish art.",
                "latitude": 53.3414,
                "longitude": -6.2532,
                "country": "Ireland",
                "city": "Dublin",
                "category": "Museum"
            },
            "landmark_detail": {
                "opening_hours": {
                    "Monday": "9:15 AM - 5:30 PM",
                    "Tuesday": "9:15 AM - 5:30 PM",
                    "Wednesday": "9:15 AM - 5:30 PM",
                    "Thursday": "9:15 AM - 5:30 PM",
                    "Friday": "9:15 AM - 5:30 PM",
                    "Saturday": "9:15 AM - 5:30 PM",
                    "Sunday": "9:15 AM - 5:30 PM"
                },
                "ticket_prices": {
                    "Adult": "Free",
                    "Child": "Free"
                },
                "historical_significance": "Founded in 1854, it features works from the Middle Ages to the present.",
                "visitor_tips": "Check for special exhibitions and events.",
                "accessibility_info": "Fully accessible."
            }
        },
        {
            "landmark": {
                "name": "Basilica Cistern",
                "description": "An ancient underground water reservoir in Istanbul, known for its massive columns.",
                "latitude": 41.0085,
                "longitude": 28.9790,
                "country": "Turkey",
                "city": "Istanbul",
                "category": "Historical Site"
            },
            "landmark_detail": {
                "opening_hours": {
                    "Monday": "9:00 AM - 5:30 PM",
                    "Tuesday": "9:00 AM - 5:30 PM",
                    "Wednesday": "9:00 AM - 5:30 PM",
                    "Thursday": "9:00 AM - 5:30 PM",
                    "Friday": "9:00 AM - 5:30 PM",
                    "Saturday": "9:00 AM - 5:30 PM",
                    "Sunday": "9:00 AM - 5:30 PM"
                },
                "ticket_prices": {
                    "Adult": "₺30",
                    "Child": "₺15"
                },
                "historical_significance": "Built in the 6th century, it held the water supply for the Great Palace.",
                "visitor_tips": "Visit early to avoid crowds.",
                "accessibility_info": "Partially accessible."
            }
        },
        {
            "landmark": {
                "name": "Topkapi Palace",
                "description": "A grand palace that was the residence of Ottoman sultans for centuries.",
                "latitude": 41.0115,
                "longitude": 28.9847,
                "country": "Turkey",
                "city": "Istanbul",
                "category": "Historical Site"
            },
            "landmark_detail": {
                "opening_hours": {
                    "Monday": "Closed",
                    "Tuesday": "9:00 AM - 6:00 PM",
                    "Wednesday": "9:00 AM - 6:00 PM",
                    "Thursday": "9:00 AM - 6:00 PM",
                    "Friday": "9:00 AM - 6:00 PM",
                    "Saturday": "9:00 AM - 6:00 PM",
                    "Sunday": "9:00 AM - 6:00 PM"
                },
                "ticket_prices": {
                    "Adult": "₺200",
                    "Child": "₺100"
                },
                "historical_significance": "Built in the 15th century, it was the administrative center of the Ottoman Empire.",
                "visitor_tips": "Purchase tickets online to avoid long queues.",
                "accessibility_info": "Partially accessible."
            }
        },
        {
            "landmark": {
                "name": "Galata Tower",
                "description": "A medieval stone tower that offers panoramic views of Istanbul.",
                "latitude": 41.0255,
                "longitude": 28.9744,
                "country": "Turkey",
                "city": "Istanbul",
                "category": "Historical Site"
            },
            "landmark_detail": {
                "opening_hours": {
                    "Monday": "9:00 AM - 8:00 PM",
                    "Tuesday": "9:00 AM - 8:00 PM",
                    "Wednesday": "9:00 AM - 8:00 PM",
                    "Thursday": "9:00 AM - 8:00 PM",
                    "Friday": "9:00 AM - 8:00 PM",
                    "Saturday": "9:00 AM - 8:00 PM",
                    "Sunday": "9:00 AM - 8:00 PM"
                },
                "ticket_prices": {
                    "Adult": "₺100",
                    "Child": "₺50"
                },
                "historical_significance": "Originally built in 1348, it was used as a watchtower.",
                "visitor_tips": "Visit at sunset for stunning views.",
                "accessibility_info": "Not wheelchair accessible."
            }
        },
        {
            "landmark": {
                "name": "Sultan Ahmed Mosque (Blue Mosque)",
                "description": "A historic mosque known for its stunning blue tiles and architectural grandeur.",
                "latitude": 41.0055,
                "longitude": 28.9768,
                "country": "Turkey",
                "city": "Istanbul",
                "category": "Religious Building"
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
                "historical_significance": "Completed in 1616, it is one of the most iconic landmarks in Istanbul.",
                "visitor_tips": "Dress modestly and visit during non-prayer times.",
                "accessibility_info": "Partially accessible."
            }
        },
        {
            "landmark": {
                "name": "Dolmabahçe Palace",
                "description": "A lavish palace that served as the main administrative center of the Ottoman Empire.",
                "latitude": 41.0392,
                "longitude": 29.0003,
                "country": "Turkey",
                "city": "Istanbul",
                "category": "Historical Site"
            },
            "landmark_detail": {
                "opening_hours": {
                    "Monday": "9:00 AM - 4:00 PM",
                    "Tuesday": "9:00 AM - 4:00 PM",
                    "Wednesday": "9:00 AM - 4:00 PM",
                    "Thursday": "9:00 AM - 4:00 PM",
                    "Friday": "9:00 AM - 4:00 PM",
                    "Saturday": "9:00 AM - 4:00 PM",
                    "Sunday": "Closed"
                },
                "ticket_prices": {
                    "Adult": "₺100",
                    "Child": "₺50"
                },
                "historical_significance": "Completed in 1856, it combines European architectural styles.",
                "visitor_tips": "Join a guided tour for in-depth history.",
                "accessibility_info": "Partially accessible."
            }
        },
        {
            "landmark": {
                "name": "Chora Church",
                "description": "An ancient church famous for its stunning mosaics and frescoes.",
                "latitude": 41.0234,
                "longitude": 28.9499,
                "country": "Turkey",
                "city": "Istanbul",
                "category": "Religious Building"
            },
            "landmark_detail": {
                "opening_hours": {
                    "Monday": "9:00 AM - 5:00 PM",
                    "Tuesday": "9:00 AM - 5:00 PM",
                    "Wednesday": "9:00 AM - 5:00 PM",
                    "Thursday": "9:00 AM - 5:00 PM",
                    "Friday": "9:00 AM - 5:00 PM",
                    "Saturday": "9:00 AM - 5:00 PM",
                    "Sunday": "9:00 AM - 5:00 PM"
                },
                "ticket_prices": {
                    "Adult": "₺30",
                    "Child": "₺15"
                },
                "historical_significance": "Originally built in the 5th century, it is known for its exquisite artwork.",
                "visitor_tips": "Look for the detailed mosaics on the walls.",
                "accessibility_info": "Not fully accessible."
            }
        },
        {
            "landmark": {
                "name": "Taksim Square",
                "description": "A major public space in Istanbul known for its cultural significance and vibrant atmosphere.",
                "latitude": 41.0369,
                "longitude": 28.9852,
                "country": "Turkey",
                "city": "Istanbul",
                "category": "Public Square"
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
                "historical_significance": "The site of many significant events in Turkish history.",
                "visitor_tips": "Enjoy the nearby shops and cafes.",
                "accessibility_info": "Accessible pathways."
            }
        },
        {
            "landmark": {
                "name": "Beylerbeyi Palace",
                "description": "A beautiful summer palace located on the Asian side of Istanbul.",
                "latitude": 41.0604,
                "longitude": 29.0207,
                "country": "Turkey",
                "city": "Istanbul",
                "category": "Historical Site"
            },
            "landmark_detail": {
                "opening_hours": {
                    "Monday": "Closed",
                    "Tuesday": "9:00 AM - 4:00 PM",
                    "Wednesday": "9:00 AM - 4:00 PM",
                    "Thursday": "9:00 AM - 4:00 PM",
                    "Friday": "9:00 AM - 4:00 PM",
                    "Saturday": "9:00 AM - 4:00 PM",
                    "Sunday": "9:00 AM - 4:00 PM"
                },
                "ticket_prices": {
                    "Adult": "₺50",
                    "Child": "₺25"
                },
                "historical_significance": "Built in the 19th century, it was used as a summer residence for sultans.",
                "visitor_tips": "Explore the beautiful gardens.",
                "accessibility_info": "Partially accessible."
            }
        },
        {
            "landmark": {
                "name": "Spice Bazaar",
                "description": "A vibrant market known for its spices, sweets, and local delicacies.",
                "latitude": 41.0244,
                "longitude": 28.9764,
                "country": "Turkey",
                "city": "Istanbul",
                "category": "Market"
            },
            "landmark_detail": {
                "opening_hours": {
                    "Monday": "9:00 AM - 7:00 PM",
                    "Tuesday": "9:00 AM - 7:00 PM",
                    "Wednesday": "9:00 AM - 7:00 PM",
                    "Thursday": "9:00 AM - 7:00 PM",
                    "Friday": "9:00 AM - 7:00 PM",
                    "Saturday": "9:00 AM - 7:00 PM",
                    "Sunday": "9:00 AM - 7:00 PM"
                },
                "ticket_prices": {
                    "Adult": "Free",
                    "Child": "Free"
                },
                "historical_significance": "Built in the 17th century, it is one of the oldest bazaars in Istanbul.",
                "visitor_tips": "Bargain for the best prices.",
                "accessibility_info": "Accessible pathways."
            }
        },
        {
            "landmark": {
                "name": "Pera Museum",
                "description": "An art museum showcasing a collection of Orientalist art.",
                "latitude": 41.0329,
                "longitude": 28.9775,
                "country": "Turkey",
                "city": "Istanbul",
                "category": "Museum"
            },
            "landmark_detail": {
                "opening_hours": {
                    "Monday": "Closed",
                    "Tuesday": "10:00 AM - 6:00 PM",
                    "Wednesday": "10:00 AM - 6:00 PM",
                    "Thursday": "10:00 AM - 6:00 PM",
                    "Friday": "10:00 AM - 6:00 PM",
                    "Saturday": "10:00 AM - 6:00 PM",
                    "Sunday": "10:00 AM - 6:00 PM"
                },
                "ticket_prices": {
                    "Adult": "₺50",
                    "Child": "₺25"
                },
                "historical_significance": "Established in 2005, it focuses on 19th-century art.",
                "visitor_tips": "Check for temporary exhibitions.",
                "accessibility_info": "Partially accessible."
            }
        },
        {
            "landmark": {
                "name": "Kilmainham Gaol",
                "description": "A historic former prison known for its role in Irish history.",
                "latitude": 53.3419,
                "longitude": -6.3032,
                "country": "Ireland",
                "city": "Dublin",
                "category": "Historical Site"
            },
            "landmark_detail": {
                "opening_hours": {
                    "Monday": "9:30 AM - 6:00 PM",
                    "Tuesday": "9:30 AM - 6:00 PM",
                    "Wednesday": "9:30 AM - 6:00 PM",
                    "Thursday": "9:30 AM - 6:00 PM",
                    "Friday": "9:30 AM - 6:00 PM",
                    "Saturday": "9:30 AM - 6:00 PM",
                    "Sunday": "9:30 AM - 6:00 PM"
                },
                "ticket_prices": {
                    "Adult": "€8",
                    "Child": "€4"
                },
                "historical_significance": "A key site in Irish nationalism and independence movements.",
                "visitor_tips": "Book tickets in advance to avoid disappointment.",
                "accessibility_info": "Partially accessible."
            }
        },
    ])


    
    process_landmarks(landmarks)