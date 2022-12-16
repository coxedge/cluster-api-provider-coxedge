# Steps for publishing a Release

### Tag creation

- Once you have ensured that your preferred release branch contains the latest code, create a tag.
```shell
git tag -a <tag-name> -m "<tag-message>"
```
NOTE: Please use `-a` while creating tag. '-a' refers to the tag being annotated. Only annotated tags can be pushed onto remote.

- Push the created tag onto remote.
```shell
git push origin <tag-name>
```

NOTE: No two tags can have the same name. In case you want to delete a tag, use `git tag -d <tag-name>`

### Release publish

- Move over to the 'Releases' tab on your remote.
- Select 'Draft a new Release'.
- Choose your created tag from the dropdown list.
    NOTE: In case you want to create a tag at this step only, start by giving a tag name and then selecting the preferred branch for the release)
- Give a Release Title.
- If you want to show your full changelog in your release, select any previous release from the Previous Tag dropdown list.
- Select Generate Release Notes and make changes according to your preference.
- Attach the binaries/files necessary for the release. In our case `metadata.yaml` and `infrastructure-components.yaml` from the generated build/releases/infrastructure-cox/latest directory our needed.
- Select if you want the release to be a Pre-Release or the Latest one.
- Hit 'Publish Release' and we now have a new release.
