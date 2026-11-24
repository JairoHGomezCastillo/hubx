# Regression smoke of the platform (not a backlog story).
@smoke
Feature: Platform health
  The gateway answers, proxies /api to the API, and the web app loads through it.

  Scenario: The gateway is up
    When I request "/health" from the gateway
    Then the response status is 200
    And the response body is {"status":"ok","service":"hubx-gw"}

  Scenario: The API answers through the gateway
    When I request "/api/health" from the gateway
    Then the response status is 200
    And the response body is {"status":"ok","service":"hubx-api"}

  @e2e
  Scenario: The web app reaches the API through its own origin
    When I open the web app
    Then the page can call "/api/health" and gets {"status":"ok","service":"hubx-api"}
