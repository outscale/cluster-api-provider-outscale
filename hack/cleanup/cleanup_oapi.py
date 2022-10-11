#!/usr/bin/python

import argparse
from datetime import datetime, timedelta
import os
from osc_sdk_python import Gateway as OSCGateway

def parseArgs():
    """Read Cli args"""
    parser = argparse.ArgumentParser(description="Parse command line args")
    parser.add_argument('--days', action='store', default=30, type=int, help='Clean up omi older than X days (default = 30 days)'),
    parser.add_argument('--owner', action='store', default='123456789123', type=str, help='Outscale account owner ID (fictional default account number)'),
    parser.add_argument('--imageId', action='store', default='ami-00000000', type=str, help='Outscale Image Id',)
    parser.add_argument('--endpointUrl', action='store', default='https://fcu.eu-west-2.outscale.com', type=str, help='Outscale Endpoint Url',)
    parser.add_argument('--imageName', action='store', default='debian123', type=str, help='Outscale Image Name',)
    return parser.parse_args()

def checkImageIdAttached(gateway, imageId, accountOwnerImageId):
    """Check Image Id"""
    vms = gateway.ReadVms().get('Vms', [])
    vmsImageId = [vm["ImageId"] for vm in vms if 'ReservationId' in vm]
    for vmImageId in vmsImageId:
        if vmImageId == imageId:
            print("Get Vm Image {}".format(imageId))
            return True
    return False

def deleteImageId(gateway, imageId, accountOwnerImageId):
    """Delete Image Id """
    res = gateway.DeleteImage(ImageId=imageId)
    if "Errors" in res:
        print("Can not delete Image {} because {}".format(imageId, res))
    print("Delete ImageId {}".format(imageId))

        
def getImageId(gateway, imageName, accountOwnerImageId):
    """Get Image Id"""
    for image in gateway.ReadImages(Filters={'ImageNames': [imageName], 'AccountIds': [accountOwnerImageId]}).get('Images', []):
        imageId = image['ImageId']
        print("Get ImageId {}".format(imageId))
        return imageId

def checkImageIdExist(gateway, imageId, accountOwnerImageId):
    """Check Image Id exist"""
    for image in gateway.ReadImages(Filters={'AccountIds': [accountOwnerImageId]}).get('Images', []):
        if image['ImageId'] == imageId:
            print("ImageId {} does exist".format(imageId))
            return True
    print("ImageId {} does not exist".format(imageId))
    return False
    
def deleteImage(gateway, accountOwnerImageId, daysCount):
    """Get Image Date"""
    for image in gateway.ReadImages(Filters={'AccountIds': [accountOwnerImageId]}).get('Images', []):
        imageId = image['ImageId']
        creationDate = image['CreationDate']
        print("Get ImageId {} in {}".format(imageId, creationDate))
        checkImageToBeDelete(gateway, accountOwnerImageId, creationDate, imageId, daysCount)

def deleteUnattachedImage(gateway, imageId, accountOwnerImageId):
    """Delete Unattahed Image"""
    imageIdAttached = checkImageIdAttached(gateway, imageId, accountOwnerImageId)
    if imageIdAttached:
        print("Can not delete imageId {} because at least one vm use this image".format(imageId))
    else:
        print("Delete ImageId {}".format(imageId))
        deleteImageId(gateway, imageId, accountOwnerImageId)

def checkImageToBeDelete(gateway, accountOwnerImageId, creationDate, imageId, daysCount):
    """Check Image DateTime"""
    omiAgeLimit = datetime.now() - timedelta(days=daysCount)
    imageDate = datetime.strptime(creationDate, '%Y-%m-%dT%H:%M:%S.%fZ')
    if imageDate < omiAgeLimit:
        print("ImageId {} will be deleted because its is more than {} days".format(imageId, daysCount))
        deleteUnattachedImage(gateway, imageId, accountOwnerImageId)
    else:
        print("ImageId {} will be saved because its is less than {} days".format(imageId, daysCount))

    
def main():
    """Main function to delete old image
    """
    access_key = os.environ['OSC_ACCESS_KEY']
    secret_key = os.environ['OSC_SECRET_KEY']
    region = os.environ['OSC_REGION']

    args = parseArgs()
    daysCount = args.days
    imageId = args.imageId
    accountOwnerImageId = args.owner
    imageName = args.imageName
    print("Cleanup image")
    gateway = OSCGateway(
        **{"access_key": access_key, "secret_key": secret_key, "region": region}
    )
    if imageName != 'debian123':
        imageId = getImageId(gateway, imageName, accountOwnerImageId)
    if imageId != 'ami-00000000':
        print("Image {} with id {} will be checked".format(imageName, imageId))
        checkImageId = checkImageIdExist(gateway, imageId, accountOwnerImageId)
        if checkImageId:
            deleteUnattachedImage(gateway, imageId, accountOwnerImageId) 
    else:
        deleteImage(gateway, accountOwnerImageId, daysCount)


if __name__ == "__main__":
    main()
