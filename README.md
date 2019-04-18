# Auth0-Demo

This is a demo that, given a defined [Auth0](https:/auth0.com) application,
will pull the identity for a given social provider, based on the Auth0 user ID,
which have the name of the provider as the first part of the user ID. (E.g.,
Google OAuth user IDs start with `google-oauth2`. The user's config is saved in
a local gob-encoded file, since the intent is to capture the refresh token app
immediately after the user consents to allow the application access.

It's okay if none of that really makes sense: at this point, this is not
specifically intended to be useful to anyone else, since this is the byproduct
of research being done to add additional authn/authz options to another
project.  This is just a demo right now and doesn't do anything useful. You
probably shouldn't store refresh tokens of any value in a file that's readable
by anyone other than the owner. You probably ought not to store refresh tokens
on the filesystem anyway. The point is, this is just a demo, and a work in
progress at that.

# Notes

Use `.env.sample` to create `.env` populated with the appropriate values,
as described in the sample file, if you want to run the server with make.

# To Do

* Hook the webserver back up.
* Serve a login page that allows the user to authenticate and authorize.
* Have the callback page make a request to the server with the user ID that
  the server will use to pull the config from Auth0.
