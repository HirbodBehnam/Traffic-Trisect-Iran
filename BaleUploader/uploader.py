import sys
import time
from selenium import webdriver
from selenium.webdriver.common.by import By
from selenium.webdriver.common.action_chains import ActionChains
import os

JS_DROP_FILES = "var k=arguments,d=k[0],g=k[1],c=k[2],m=d.ownerDocument||document;for(var e=0;;){var f=d.getBoundingClientRect(),b=f.left+(g||(f.width/2)),a=f.top+(c||(f.height/2)),h=m.elementFromPoint(b,a);if(h&&d.contains(h)){break}if(++e>1){var j=new Error('Element not interactable');j.code=15;throw j}d.scrollIntoView({behavior:'instant',block:'center',inline:'center'})}var l=m.createElement('INPUT');l.setAttribute('type','file');l.setAttribute('multiple','');l.setAttribute('style','position:fixed;z-index:2147483647;left:0;top:0;');l.onchange=function(q){l.parentElement.removeChild(l);q.stopPropagation();var r={constructor:DataTransfer,effectAllowed:'all',dropEffect:'none',types:['Files'],files:l.files,setData:function u(){},getData:function o(){},clearData:function s(){},setDragImage:function i(){}};if(window.DataTransferItemList){r.items=Object.setPrototypeOf(Array.prototype.map.call(l.files,function(x){return{constructor:DataTransferItem,kind:'file',type:x.type,getAsFile:function v(){return x},getAsString:function y(A){var z=new FileReader();z.onload=function(B){A(B.target.result)};z.readAsText(x)},webkitGetAsEntry:function w(){return{constructor:FileSystemFileEntry,name:x.name,fullPath:'/'+x.name,isFile:true,isDirectory:false,file:function z(A){A(x)}}}}}),{constructor:DataTransferItemList,add:function t(){},clear:function p(){},remove:function n(){}})}['dragenter','dragover','drop'].forEach(function(v){var w=m.createEvent('DragEvent');w.initMouseEvent(v,true,true,m.defaultView,0,0,0,b,a,false,false,false,false,0,null);Object.setPrototypeOf(w,null);w.dataTransfer=r;Object.setPrototypeOf(w,DragEvent.prototype);h.dispatchEvent(w)})};m.documentElement.appendChild(l);l.getBoundingClientRect();return l"

def drop_files(element, files, offsetX=0, offsetY=0):
    driver = element.parent
    isLocal = not driver._is_remote or '127.0.0.1' in driver.command_executor._url
    paths = []
    
    # ensure files are present, and upload to the remote server if session is remote
    for file in (files if isinstance(files, list) else [files]) :
        if not os.path.isfile(file) :
            raise FileNotFoundError(file)
        paths.append(file if isLocal else element._upload(file))
    
    value = '\n'.join(paths)
    elm_input = driver.execute_script(JS_DROP_FILES, element, offsetX, offsetY)
    elm_input._execute('sendKeysToElement', {'value': [value], 'text': value})

# Validate each file
if len(sys.argv) <= 1:
    print("Please pass at least one file to upload")
    exit(1)
for file in sys.argv[1:]:
    try:
        if os.path.getsize(file) > 500_000_000:
            print("File", file, "is too large")
            exit(1)
    except Exception as e:
        print("Cannot open file", file, ":", e)
        exit(1)

options = webdriver.ChromeOptions()
options.add_argument("--user-data-dir=./userdata/")
with webdriver.Chrome(options=options) as driver:
    # Load the page
    driver.get("https://web.bale.ai")
    time.sleep(5)
    # Find the chat
    chats = driver.find_elements(By.XPATH, "//div[contains(text(), 'فضای شخصی')]")
    print("Find chats:", len(chats))
    ac = ActionChains(driver)
    ac.move_to_element(chats[0]).click().perform()
    time.sleep(5)
    # Upload each file
    for file in sys.argv[1:]:
        dropzones = driver.find_elements(By.CLASS_NAME, "main-section-container")
        print("Drop zone candidates:", len(dropzones))
        drop_files(dropzones[0], file)
        time.sleep(5)
        # Click on confirm
        send_button = driver.find_elements(By.XPATH, "//button[contains(text(), 'ارسال')]")
        print("Send button candidates:", len(send_button))
        ac = ActionChains(driver)
        ac.move_to_element(send_button[0]).click().perform()
        # Wait for upload
        time.sleep(50)

print("File upload done!")
