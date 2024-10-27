import hashlib
import geopy
import requests
from bs4 import BeautifulSoup
import json
import os
from datetime import datetime
import pandas as pd
import spacy
from sqlalchemy import create_engine, Column, String, Float, JSON, Integer, DateTime, Boolean
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.orm import sessionmaker
import logging
from urllib.parse import urljoin
import time
from fake_useragent import UserAgent
import wikipedia
from geopy.geocoders import Nominatim
from PIL import Image
from io import BytesIO
import concurrent.futures
import nltk
from nltk.tokenize import word_tokenize
from nltk.tag import pos_tag
from nltk.chunk import ne_chunk
from transformers import pipeline
import validators
from schema import Schema, And, Use, Optional
import googlemaps
from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
import re
from dateutil.parser import parse
import numpy as np
from deep_translator import GoogleTranslator
import torch
from timeout_decorator import timeout

# Download required NLTK data
nltk.download('averaged_perceptron_tagger')
nltk.download('maxent_ne_chunker')
nltk.download('words')

# Set up logging with rotation
import logging.handlers
log_handler = logging.handlers.RotatingFileHandler(
    'landmark_scraper.log',
    maxBytes=1024*1024,
    backupCount=5
)
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[log_handler]
)

# Database setup
Base = declarative_base()

class Landmark(Base):
    __tablename__ = 'landmarks'
    
    id = Column(Integer, primary_key=True)
    name = Column(String, unique=True)
    description = Column(String)
    latitude = Column(Float)
    longitude = Column(Float)
    country = Column(String)
    city = Column(String)
    category = Column(String)
    landmark_detail = Column(JSON)
    image_paths = Column(JSON)  # Store multiple image paths
    last_updated = Column(DateTime, default=datetime.utcnow)
    data_sources = Column(JSON)  # Track where data came from
    validation_status = Column(Boolean, default=False)

# Data validation schemas
landmark_schema = Schema({
    'name': And(str, len),
    'description': And(str, len),
    'latitude': And(float, lambda n: -90 <= n <= 90),
    'longitude': And(float, lambda n: -180 <= n <= 180),
    'country': And(str, len),
    'city': And(str, len),
    'category': And(str, len),
    Optional('image_paths'): [str],
    Optional('data_sources'): dict,
})

class DataSourceManager:
    """Manages multiple data sources for landmark information"""
    
    def __init__(self, api_keys):
        self.gmaps = googlemaps.Client(key=api_keys['google_maps'])
        self.setup_selenium()
        self.translator = GoogleTranslator(source='auto', target='en')
        
    def setup_selenium(self):
        """Set up headless Chrome browser"""
        chrome_options = Options()
        chrome_options.add_argument("--headless")
        chrome_options.add_argument("--no-sandbox")
        chrome_options.add_argument("--disable-dev-shm-usage")
        self.driver = webdriver.Chrome(options=chrome_options)

    def get_tripadvisor_data(self, landmark_name):
        """Scrape TripAdvisor for landmark information"""
        try:
            search_url = f"https://www.tripadvisor.com/Search?q={landmark_name}"
            self.driver.get(search_url)
            
            # Wait for search results
            wait = WebDriverWait(self.driver, 10)
            first_result = wait.until(EC.presence_of_element_located((By.CSS_SELECTOR, ".result-title")))
            first_result.click()
            
            # Extract information
            opening_hours = self.extract_tripadvisor_hours()
            prices = self.extract_tripadvisor_prices()
            reviews = self.extract_tripadvisor_reviews()
            
            return {
                'opening_hours': opening_hours,
                'prices': prices,
                'reviews': reviews
            }
        except Exception as e:
            logging.error(f"Error scraping TripAdvisor for {landmark_name}: {str(e)}")
            return None

    @timeout(30)
    def extract_tripadvisor_hours(self):
        """Extract opening hours from TripAdvisor page"""
        try:
            hours_button = WebDriverWait(self.driver, 5).until(
                EC.presence_of_element_located((By.CSS_SELECTOR, "[data-tab='TABS_HOURS']"))
            )
            hours_button.click()
            
            hours_elements = self.driver.find_elements(By.CSS_SELECTOR, ".hours_text")
            hours = {}
            for element in hours_elements:
                day, time = element.text.split(": ")
                hours[day.strip()] = time.strip()
            return hours
        except:
            return None

    def extract_tripadvisor_prices(self):
        """Extract ticket prices from TripAdvisor page"""
        try:
            price_elements = self.driver.find_elements(By.CSS_SELECTOR, ".price_text")
            prices = {}
            for element in price_elements:
                category, price = element.text.split(": ")
                prices[category.strip()] = price.strip()
            return prices
        except:
            return None

    def extract_tripadvisor_reviews(self):
        """Extract recent reviews from TripAdvisor"""
        try:
            reviews = []
            review_elements = self.driver.find_elements(By.CSS_SELECTOR, ".review-container")[:5]
            for element in review_elements:
                review = {
                    'text': element.find_element(By.CSS_SELECTOR, ".review-text").text,
                    'rating': element.find_element(By.CSS_SELECTOR, ".rating-circle").text,
                    'date': element.find_element(By.CSS_SELECTOR, ".review-date").text
                }
                reviews.append(review)
            return reviews
        except:
            return []

    def get_google_places_data(self, landmark_name):
        """Get data from Google Places API"""
        try:
            place_result = self.gmaps.places(landmark_name)
            if place_result['status'] == 'OK':
                place_id = place_result['results'][0]['place_id']
                place_details = self.gmaps.place(place_id, fields=[
                    'name', 'formatted_address', 'geometry', 'opening_hours',
                    'price_level', 'rating', 'reviews', 'photos'
                ])
                
                return {
                    'google_data': place_details['result'],
                    'photos': self.get_google_photos(place_details['result'].get('photos', []))
                }
        except Exception as e:
            logging.error(f"Error getting Google Places data for {landmark_name}: {str(e)}")
        return None

    def get_google_photos(self, photo_references, max_photos=5):
        """Download photos from Google Places API"""
        photos = []
        for ref in photo_references[:max_photos]:
            try:
                photo_url = f"https://maps.googleapis.com/maps/api/place/photo?maxwidth=800&photoreference={ref['photo_reference']}&key={self.gmaps._api_key}"
                photos.append(photo_url)
            except Exception as e:
                logging.error(f"Error downloading Google photo: {str(e)}")
        return photos

class CategoryDetector:
    """Detects landmark categories using NLP"""
    
    def __init__(self):
        self.classifier = pipeline("zero-shot-classification")
        self.categories = [
            "Historical Site",
            "Museum",
            "Religious Site",
            "Natural Wonder",
            "Archaeological Site",
            "Modern Architecture",
            "Palace",
            "Castle",
            "Monument",
            "Park or Garden"
        ]

    def detect_category(self, description):
        """Detect landmark category from description"""
        try:
            result = self.classifier(description, self.categories)
            return result['labels'][0]  # Return highest confidence category
        except Exception as e:
            logging.error(f"Error detecting category: {str(e)}")
            return "Unknown"

    def extract_entities(self, text):
        """Extract named entities from text"""
        tokens = word_tokenize(text)
        tagged = pos_tag(tokens)
        entities = ne_chunk(tagged)
        return entities

class DataValidator:
    """Validates and cleanses landmark data"""
    
    @staticmethod
    def validate_coordinates(lat, lon):
        return isinstance(lat, (int, float)) and isinstance(lon, (int, float)) and \
               -90 <= lat <= 90 and -180 <= lon <= 180

    @staticmethod
    def validate_url(url):
        return bool(validators.url(url))

    @staticmethod
    def validate_date_string(date_str):
        try:
            parse(date_str)
            return True
        except:
            return False

    @staticmethod
    def clean_text(text):
        """Clean and normalize text content"""
        if not text:
            return ""
        # Remove extra whitespace
        text = re.sub(r'\s+', ' ', text.strip())
        # Remove special characters
        text = re.sub(r'[^\w\s.,!?-]', '', text)
        return text

    @staticmethod
    def validate_price(price_str):
        """Validate and normalize price strings"""
        if not price_str:
            return None
        # Extract number and currency
        match = re.search(r'([€$£¥])?(\d+(?:\.\d{2})?)', price_str)
        if match:
            currency, amount = match.groups()
            return f"{currency or '€'}{amount}"
        return None

class EnhancedLandmarkScraper:
    def __init__(self, api_keys):
        self.ua = UserAgent()
        self.session = requests.Session()
        self.geolocator = Nominatim(user_agent="landmark_scraper")
        self.engine = create_engine('postgresql://postgres:tmiandwVOjwJXjCVGmxyoMeAKQjkRGrl@junction.proxy.rlwy.net:27034/railway')
        Base.metadata.create_all(self.engine)
        self.Session = sessionmaker(bind=self.engine)
        self.image_dir = "landmark_images"
        self.data_source_manager = DataSourceManager(api_keys)
        self.category_detector = CategoryDetector()
        self.validator = DataValidator()
        
        # Create image directory if it doesn't exist
        os.makedirs(self.image_dir, exist_ok=True)

    def get_wikipedia_info(self, landmark_name):
       
        try:
            # Search Wikipedia for the landmark
            try:
                wiki_page = wikipedia.page(landmark_name, auto_suggest=True)
            except wikipedia.DisambiguationError as e:
                # If disambiguation page, try the first suggested page
                wiki_page = wikipedia.page(e.options[0])
            except wikipedia.PageError:
                logging.error(f"Wikipedia page not found for {landmark_name}")
                return None

            # Extract main image URL
            image_url = None
            if wiki_page.images:
                # Filter out SVG and low-quality images
                valid_images = [img for img in wiki_page.images 
                              if any(ext in img.lower() for ext in ['.jpg', '.jpeg', '.png']) 
                              and 'logo' not in img.lower() 
                              and 'icon' not in img.lower()]
                if valid_images:
                    image_url = valid_images[0]

            # Get the summary and clean it
            summary = self.validator.clean_text(wiki_page.summary)

            # Extract coordinates if available
            coordinates = None
            try:
                coords = self.geolocator.geocode(landmark_name)
                if coords:
                    coordinates = {
                        'latitude': coords.latitude,
                        'longitude': coords.longitude
                    }
            except Exception as e:
                logging.warning(f"Could not get coordinates for {landmark_name}: {str(e)}")

            # Get location information
            location_info = self.extract_location_info(wiki_page.content)

            # Compile all information
            wiki_data = {
                'title': wiki_page.title,
                'description': summary,
                'image_url': image_url,
                'url': wiki_page.url,
                'coordinates': coordinates,
                'country': location_info.get('country'),
                'city': location_info.get('city'),
                'references': wiki_page.references,
                'last_updated': datetime.utcnow().isoformat()
            }

            logging.info(f"Successfully retrieved Wikipedia data for {landmark_name}")
            return wiki_data

        except Exception as e:
            logging.error(f"Error getting Wikipedia info for {landmark_name}: {str(e)}")
            return None

    def extract_location_info(self, content):
        """
        Extract country and city information from Wikipedia content
        
        Args:
            content (str): Wikipedia page content
            
        Returns:
            dict: Dictionary containing country and city information
        """
        location_info = {'country': None, 'city': None}
        
        # Common patterns for location information
        country_patterns = [
            r'located in (?:the )?([A-Za-z\s]+)',
            r'(?:is|was) a [^.]+? in (?:the )?([A-Za-z\s]+)',
            r'(?:is|was) an? [^.]+? in (?:the )?([A-Za-z\s]+)'
        ]
        
        city_patterns = [
            r'in ([A-Za-z\s]+)(?:,|\s+is)',
            r'located in ([A-Za-z\s]+),',
            r'situated in ([A-Za-z\s]+),'
        ]

        # Try to find country
        for pattern in country_patterns:
            match = re.search(pattern, content)
            if match:
                country = match.group(1).strip()
                # Validate that it's actually a country name
                try:
                    country_check = self.geolocator.geocode(country)
                    if country_check and 'country' in country_check.raw.get('type', '').lower():
                        location_info['country'] = country
                        break
                except:
                    continue

        # Try to find city
        for pattern in city_patterns:
            match = re.search(pattern, content)
            if match:
                city = match.group(1).strip()
                # Validate that it's actually a city name
                try:
                    city_check = self.geolocator.geocode(city)
                    if city_check and 'city' in city_check.raw.get('type', '').lower():
                        location_info['city'] = city
                        break
                except:
                    continue

        return location_info

    def process_landmark(self, landmark_name):
        """Process a single landmark"""
        try:
            # Get data from multiple sources
            wiki_data = self.get_wikipedia_info(landmark_name)
            google_data = self.data_source_manager.get_google_places_data(landmark_name)
            tripadvisor_data = self.data_source_manager.get_tripadvisor_data(landmark_name)
            
            if not wiki_data and not google_data:
                logging.error(f"No data found for {landmark_name}")
                return None

            # Combine and validate data
            combined_data = self.combine_data_sources(
                landmark_name,
                wiki_data,
                google_data,
                tripadvisor_data
            )

            # Validate combined data
            try:
                landmark_schema.validate(combined_data['landmark'])
            except Exception as e:
                logging.error(f"Data validation failed for {landmark_name}: {str(e)}")
                return None

            # Save to database
            self.save_to_database(combined_data)
            
            return combined_data

        except Exception as e:
            logging.error(f"Error processing {landmark_name}: {str(e)}")
            return None
        
        

    def combine_data_sources(self, landmark_name, wiki_data, google_data, tripadvisor_data):
        """Combine and normalize data from multiple sources"""
        combined = {
            'landmark': {
                'name': landmark_name,
                'description': '',
                'latitude': None,
                'longitude': None,
                'country': '',
                'city': '',
                'category': 'Unknown',
                'image_paths': [],
                'data_sources': {}
            },
            'landmark_detail': {
                'opening_hours': {},
                'ticket_prices': {},
                'reviews': [],
                'accessibility_info': '',
                'visitor_tips': []
            }
        }

        # Combine Wikipedia data
        if wiki_data:
            combined['landmark']['description'] = self.validator.clean_text(wiki_data.get('description', ''))
            if wiki_data.get('image_url'):
                combined['landmark']['image_paths'].append(wiki_data['image_url'])
            combined['landmark']['data_sources']['wikipedia'] = True

        # Combine Google Places data
        if google_data:
            google_info = google_data['google_data']
            combined['landmark']['latitude'] = google_info['geometry']['location']['lat']
            combined['landmark']['longitude'] = google_info['geometry']['location']['lng']
            combined['landmark']['image_paths'].extend(google_data['photos'])
            combined['landmark_detail']['rating'] = google_info.get('rating')
            combined['landmark_detail']['reviews'].extend(google_info.get('reviews', []))
            combined['landmark']['data_sources']['google_places'] = True

        # Combine TripAdvisor data
        if tripadvisor_data:
            combined['landmark_detail']['opening_hours'].update(tripadvisor_data.get('opening_hours', {}))
            combined['landmark_detail']['ticket_prices'].update(tripadvisor_data.get('prices', {}))
            combined['landmark_detail']['reviews'].extend(tripadvisor_data.get('reviews', []))
            combined['landmark']['data_sources']['tripadvisor'] = True

        # Detect category using NLP
        if combined['landmark']['description']:
            combined['landmark']['category'] = self.category_detector.detect_category(
                combined['landmark']['description']
            )

        return combined

    def save_to_database(self, data):
        """Save landmark data to database"""
        session = self.Session()
        try:
            landmark = Landmark(
                name=data['landmark']['name'],
                description=data['landmark']['description'],
                latitude=data['landmark']['latitude'],
                longitude=data['landmark']['longitude'],
                country=data['landmark']['country'],
                city=data['landmark']['city'],
                category=data['landmark']['category'],
                landmark_detail=data['landmark_detail'],
                image_paths=data['landmark']['image_paths'],
                data_sources=data['landmark']['data_sources'],
                last_updated=datetime.utcnow(),
                validation_status=True
            )
            session.merge(landmark)  # Use merge instead of add to handle updates
            session.commit()
            logging.info(f"Successfully saved {data['landmark']['name']} to database")
        except Exception as e:
            session.rollback()
            logging.error(f"Database error for {data['landmark']['name']}: {str(e)}")
        finally:
            session.close()

    def process_landmarks_parallel(self, landmark_list, max_workers=4):
        """Process multiple landmarks in parallel"""
        results = []
        with concurrent.futures.ThreadPoolExecutor(max_workers=max_workers) as executor:
            future_to_landmark = {
                executor.submit(self.process_landmark, landmark): landmark
                for landmark in landmark_list
            }
            
            for future in concurrent.futures.as_completed(future_to_landmark):
                landmark = future_to_landmark[future]
                try:
                    data = future.result()
                    if data:
                        results.append(data)
                except Exception as e:
                    logging.error(f"Error processing {landmark}: {str(e)}")
                
        return results

class ImageProcessor:
    """Handles image downloading, processing, and storage"""
    
    def __init__(self, base_dir="landmark_images"):
        self.base_dir = base_dir
        os.makedirs(base_dir, exist_ok=True)
        
    def process_image(self, image_url, landmark_name):
        """Download, process, and save an image"""
        try:
            response = requests.get(image_url, stream=True)
            if response.status_code == 200:
                # Create unique filename
                file_hash = hashlib.md5(image_url.encode()).hexdigest()
                safe_name = re.sub(r'[^a-zA-Z0-9]', '_', landmark_name)
                filename = f"{safe_name}_{file_hash[:8]}.jpg"
                filepath = os.path.join(self.base_dir, filename)
                
                # Process and save image
                img = Image.open(BytesIO(response.content))
                img = self.optimize_image(img)
                img.save(filepath, 'JPEG', quality=85)
                
                return filepath
        except Exception as e:
            logging.error(f"Error processing image for {landmark_name}: {str(e)}")
            return None
            
    def optimize_image(self, img):
        """Optimize image for web use"""
        # Convert to RGB if necessary
        if img.mode in ('RGBA', 'P'):
            img = img.convert('RGB')
            
        # Resize if too large
        max_size = (1200, 1200)
        if img.size[0] > max_size[0] or img.size[1] > max_size[1]:
            img.thumbnail(max_size, Image.LANCZOS)
            
        return img

class DataEnricher:
    """Enriches landmark data with additional information"""
    
    def __init__(self):
        self.nlp = spacy.load('en_core_web_sm')
        
    def enrich_description(self, description):
        """Extract additional information from description"""
        doc = self.nlp(description)
        
        # Extract dates
        dates = []
        for ent in doc.ents:
            if ent.label_ == 'DATE':
                dates.append(ent.text)
                
        # Extract key phrases
        key_phrases = []
        for chunk in doc.noun_chunks:
            if len(chunk.text.split()) >= 2:
                key_phrases.append(chunk.text)
                
        return {
            'dates_mentioned': dates,
            'key_phrases': key_phrases
        }
        
    def find_related_landmarks(self, landmark_data, all_landmarks):
        """Find related landmarks based on category and location"""
        related = []
        for other in all_landmarks:
            if other['name'] != landmark_data['name']:
                # Same category
                if other['category'] == landmark_data['category']:
                    related.append({
                        'name': other['name'],
                        'relationship': 'same_category'
                    })
                    
                # Nearby (within 50km)
                if self.calculate_distance(
                    (landmark_data['latitude'], landmark_data['longitude']),
                    (other['latitude'], other['longitude'])
                ) <= 50:
                    related.append({
                        'name': other['name'],
                        'relationship': 'nearby'
                    })
                    
        return related
        
    def calculate_distance(self, coord1, coord2):
        """Calculate distance between two coordinates in kilometers"""
        return geopy.distance.distance(coord1, coord2).km

def main():
    # API keys configuration
    api_keys = {
        'google_maps': 'AIzaSyAt21cb7sQatq8oMry98Y3LE68NC87qBcs',
        # Add other API keys as needed
    }

    # Initialize scraper
    scraper = EnhancedLandmarkScraper(api_keys)
    
    # List of landmarks to scrape (can be expanded)
    landmarks_to_scrape = [
        "Eiffel Tower",
        "Colosseum",
        "Taj Mahal",
        "Great Wall of China",
        "Petra",
        "Machu Picchu",
        "Christ the Redeemer",
        "Stonehenge",
        "Acropolis of Athens",
        "Angkor Wat",
        "Pyramids of Giza",
        "Tower of London",
        "Forbidden City",
        "Saint Basil's Cathedral",
        "Sagrada Familia"
    ]

    # Process landmarks in parallel
    results = scraper.process_landmarks_parallel(landmarks_to_scrape)
    
    # Save results to JSON file as backup
    with open('landmarks_data.json', 'w', encoding='utf-8') as f:
        json.dump(results, f, ensure_ascii=False, indent=4)
        
    # Print summary
    print(f"\nProcessing complete!")
    print(f"Successfully processed: {len(results)} landmarks")
    print(f"Failed landmarks: {len(landmarks_to_scrape) - len(results)}")

if __name__ == "__main__":
    main()