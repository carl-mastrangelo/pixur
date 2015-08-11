Picture Deletion
====
Deleting pictures is uncomfortable to both server administrators and users alike.  Deletion is inherently risky, requiring significant testing to ensure that the correct data is removed while unrelated data is untouched.  Deletion also has social stigma associated resulting in a worse user experience for those in favor of keeping the image.  However, picture deletion is necessary for both technical and social reasons.  In order to address these issues, deletion is implemented with the following goals in mind:

## Goals:
- Remove low quality or other undesirable pictures.  Such pictures cause the site community to not enjoy  the site and therefore endanger the long term health of the site
- Make room on space limited servers.  Because pictures are typically large, and expected to be uploaded in bulk, servers can quickly run out of room to store them.
- Provide a commentary period before pictures are deleted.  Abrupt removal of pictures is bad user experience, and leaves no way to discuss or contest the removal 
  
## Non Goals:
-  Prevent abuse and re upload.  Abuse is better dealt with at the user level rather than the picture level.  Enforcement of this is outside the scope of this document.
-  Remove duplicate pictures or reposts.  Picture merging should be done instead.  

Deletion is split into three separate phases: Soft Deleted, Hard Deleted, and Purged.  The typical flow is from Soft Deleted into the Hard Deleted state.  Purged is reserved for special circumstances and should not be necessary.

## Soft Deletion
Soft Deletion, or "Marked for Deletion" is a phase that indicates that the picture is no longer desired.  The picture will still be listed in search results and on the index page, functioning normally as before.  Tags, comments, votes, and other metadata are persisted and can be freely changed.  Common reasons to enter this state include the picture being too small, blurry, or being against site rules.  Pictures that are hard deleted cannot be marked as soft deleted.

Soft deletion can be done with two optional parameters: a reason and a pending deletion time.  The reason field is a very brief explanation of why the picture was marked for deletion.  The pending deletion time is the time after which the picture can be automatically be hard deleted.  If the pending deletion time is not set, it will not be automatically hard deleted unless done by an administrator. 

## Hard Deletion
Hard Deletion, or just "Deleted" is the phase that actually remove the picture file.  All thumbnail images are removed in addition to the original file.  Metadata such as tags, comments, and votes are still preserved in the database, but cannot be altered by user action.  In this way, the previous picture data is kept for posterity.  The time that the picture is actually deleted is recorded on the picture, and the picture is marked as "hidden".  This means it will no longer appear on search results or the index page, and cannot be viewed without specific user action.

Note that hard deletion is not specifically a precedent setting action.  A picture may exit the hard deleted state by being uploaded again (via Merge).  This is an expected behavior if an image is deleted to save on space, but uploaded again after more space is available.  Users who abusively re upload a deleted image should either have their access limited.  Users (especially new ones) who accidentally upload an image against the rules should be gently informed of their infraction.  Such violations need a social solution rather than a technical one.
  
## Purged
Purged is similar to Hard deletion, but all database references and other metadata are removed upon purging.  After purging an image, the site will act as if it had never been uploaded.  This is currently a heavy handed approach to moderation and is currently for cleaning up the site while it is still in early development.  It is not clear that purging will be part of the future of Pixur.

## Future work
Sometimes administrators need stopgap ways of preventing images from being uploaded.  It would be a good idea for a Hard deleted pictures to record that they should not be uploaded again.  This may be necessary to prevent abuse but will not be implemented until a specific need is shown.
