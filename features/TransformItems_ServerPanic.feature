Feature: Transform Items

    Scenario: Server Panic
        When I request a GRPC method "/grpctest.ItemService/TransformItems" with payload:
        """
        []
        """

        Then I should have a GRPC response with code "Internal"
