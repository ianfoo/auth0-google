// jshint esversion: 6

const isCallback = () => !!window.location.hash;

console.log("running auth0 handler");

if (!isCallback()) {
	console.log("need to log in");
	// Get an access token and an ID token (JWT).
	// TODO what is the access token for, exactly? (Auth0 mgmt API uses ID token.)
	const AUTH_CONFIG = {
	  domain: 'metricstory.auth0.com',
	  clientId: '0YO3rhFC1cFvdcjG6vW3m0naHA5PVVvR',
	  callbackUrl: 'http://localhost:3000/callback'
	};

	let webAuth = new auth0.WebAuth({
		responseType: 'token id_token',
		domain: AUTH_CONFIG.domain,
		clientID: AUTH_CONFIG.clientId,
		redirectUri: AUTH_CONFIG.callbackUrl
	});

	// This will end up redirecting to Google directly because of the
	// "connection" property. The "prompt" property will force the
	webAuth.authorize({
		connection: 'google-oauth2',
		scope: 'openid',
		state: 'pickAMoreSecureStateThanThisYouN00b', // Nasty
		prompt: 'login',
		accessType: 'offline'
	});
} else {
	console.log("handling auth callback!");

	// Save the tokens?
	window.localStorage.setItem("auth0_access_token", "");
	window.localStorage.setItem("auth0_id_token", "");

	// Send the ID token on to the back end, in order that a Google access and refresh token
	// might be obtained for accessing Google Analytics.
}
