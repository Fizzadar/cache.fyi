# 2025-09-24 Integrate Linkwarden

Aim is to integrate content and linkwarden. 

Linwarden API docs: https://docs.linkwarden.app/api/api-introduction

- create new linkwarden_link_id field on content
- create a new content processor (@internal/workers/content_processors) that, when content type is url, creates the link in linkwarden and sets the linkwarden_link_id column
- if link id set, link to linkwarden in the content table
