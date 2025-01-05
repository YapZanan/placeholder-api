import requests
import random
import time
from concurrent.futures import ThreadPoolExecutor

# Base URL of the deployed API
BASE_URL = "https://placeholder-api-hz1q.onrender.com/placeholder"

# Function to generate random query parameters
def generate_random_params():
    width = random.randint(100, 1920)  # Random width between 100 and 1920
    height = random.randint(100, 1080)  # Random height between 100 and 1080
    text = "RandomText" + str(random.randint(1, 100))  # Random text
    font_size = random.randint(20, 50)  # Random font size between 20 and 50
    bg_color = "#{:02x}{:02x}{:02x}".format(random.randint(0, 255), random.randint(0, 255), random.randint(0, 255))  # Random RGB hex color
    font_color = "#{:02x}{:02x}{:02x}".format(random.randint(0, 255), random.randint(0, 255), random.randint(0, 255))  # Random RGB hex color
    
    params = {
        "width": width,
        "height": height,
        "text": text,
        "font_size": font_size,
        "bg_color": bg_color,
        "font_color": font_color
    }
    
    return params

def make_request(success_count, failure_count, response_times):
    params = generate_random_params()
    url = f"{BASE_URL}?width={params['width']}&height={params['height']}&text={requests.utils.quote(params['text'])}&font_size={params['font_size']}&bg_color={requests.utils.quote(params['bg_color'])}&font_color={requests.utils.quote(params['font_color'])}"
    
    start_time = time.time()
    try:
        response = requests.get(url)
        end_time = time.time()
        response_time = end_time - start_time
        if response.status_code == 200:
            success_count.append(1)
            response_times.append(response_time)
            print(f"Success: {url} - {response.status_code} - {response_time:.2f}s")
        else:
            failure_count.append(1)
            print(f"Failed: {url} - {response.status_code}")
    except requests.RequestException as e:
        end_time = time.time()
        response_time = end_time - start_time
        failure_count.append(1)
        print(f"Request failed: {url} - {str(e)} - {response_time:.2f}s")

def run_benchmark(num_requests, num_threads):
    success_count = []
    failure_count = []
    response_times = []
    
    with ThreadPoolExecutor(max_workers=num_threads) as executor:
        futures = []
        for _ in range(num_requests):
            futures.append(executor.submit(make_request, success_count, failure_count, response_times))
        
        for future in futures:
            future.result()

    return success_count, failure_count, response_times

if __name__ == "__main__":
    num_requests = 1000  # Total number of requests to send
    num_threads = 10    # Number of concurrent threads

    start_time = time.time()
    print(f"Starting benchmark with {num_requests} requests and {num_threads} threads...")
    
    success_count, failure_count, response_times = run_benchmark(num_requests, num_threads)
    
    end_time = time.time()
    elapsed_time = end_time - start_time

    success_percentage = (len(success_count) / num_requests) * 100
    failure_percentage = (len(failure_count) / num_requests) * 100
    average_response_time = sum(response_times) / len(response_times) if response_times else 0
    response_times.sort()
    one_percent_index = int(len(response_times) * 0.01)
    fastest_one_percent = response_times[:one_percent_index]

    print(f"\nBenchmark completed in {elapsed_time:.2f} seconds.")
    print(f"Total requests: {num_requests}")
    print(f"Success percentage: {success_percentage:.2f}%")
    print(f"Failure percentage: {failure_percentage:.2f}%")
    print(f"Average response time: {average_response_time:.2f} seconds")
    print(f"Fastest 1% response times: {fastest_one_percent}")
    
    print(f"\nSummary statistics:")
    print(f"Total successful requests: {len(success_count)}")
    print(f"Total failed requests: {len(failure_count)}")
    print(f"Min response time: {min(response_times) if response_times else 'N/A'}s")
    print(f"Max response time: {max(response_times) if response_times else 'N/A'}s")
